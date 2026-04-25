package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalDocumentStorage struct {
	basePath string
}

func NewLocalDocumentStorage(basePath string) *LocalDocumentStorage {
	return &LocalDocumentStorage{basePath: basePath}
}

func (s *LocalDocumentStorage) Save(ctx context.Context, id, originalFileName string, content io.Reader) (string, error) {
	if s.basePath == "" {
		return "", fmt.Errorf("document storage path is not configured")
	}

	if err := os.MkdirAll(s.basePath, 0o755); err != nil {
		return "", fmt.Errorf("create document storage directory: %w", err)
	}

	fileName := id + "_" + sanitizeFileName(originalFileName)
	path := filepath.Join(s.basePath, fileName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create stored document file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, readerWithContext{ctx: ctx, reader: content}); err != nil {
		return "", fmt.Errorf("store document file: %w", err)
	}

	return path, nil
}

func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, string(filepath.Separator), "_")
	if name == "." || name == "" {
		return "document.docx"
	}

	return name
}

type readerWithContext struct {
	ctx    context.Context
	reader io.Reader
}

func (r readerWithContext) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return r.reader.Read(p)
}
