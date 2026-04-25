package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) AddMetadata(content []byte, documentID string, date time.Time) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("open docx archive: %w", err)
	}

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	documentXMLFound := false

	for _, file := range reader.File {
		if err := copyDocxFile(writer, file, documentID, date, &documentXMLFound); err != nil {
			writer.Close()
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close updated docx archive: %w", err)
	}
	if !documentXMLFound {
		return nil, fmt.Errorf("docx document.xml not found")
	}

	return output.Bytes(), nil
}

func copyDocxFile(writer *zip.Writer, file *zip.File, documentID string, date time.Time, documentXMLFound *bool) error {
	header := file.FileHeader
	header.Name = file.Name
	header.Method = zip.Deflate

	target, err := writer.CreateHeader(&header)
	if err != nil {
		return fmt.Errorf("create docx archive entry %q: %w", file.Name, err)
	}

	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("open docx archive entry %q: %w", file.Name, err)
	}
	defer source.Close()

	if file.Name != "word/document.xml" {
		if _, err := io.Copy(target, source); err != nil {
			return fmt.Errorf("copy docx archive entry %q: %w", file.Name, err)
		}
		return nil
	}

	*documentXMLFound = true
	documentXML, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("read word/document.xml: %w", err)
	}

	updatedXML, err := addMetadataParagraph(string(documentXML), documentID, date)
	if err != nil {
		return err
	}

	if _, err := target.Write([]byte(updatedXML)); err != nil {
		return fmt.Errorf("write updated word/document.xml: %w", err)
	}

	return nil
}

func addMetadataParagraph(documentXML string, documentID string, date time.Time) (string, error) {
	bodyEnd := strings.LastIndex(documentXML, "</w:body>")
	if bodyEnd == -1 {
		return "", fmt.Errorf("docx body end tag not found")
	}

	metadata := fmt.Sprintf(
		"Document UUID: %s; Date: %s",
		documentID,
		date.UTC().Format(time.RFC3339),
	)

	return documentXML[:bodyEnd] + paragraph(metadata) + documentXML[bodyEnd:], nil
}

func paragraph(text string) string {
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(text))

	return "<w:p><w:r><w:t>" + escaped.String() + "</w:t></w:r></w:p>"
}
