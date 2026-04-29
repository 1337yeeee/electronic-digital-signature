import { ReactNode } from "react";

type SecurityNoticeProps = {
  title: string;
  children: ReactNode;
};

export function SecurityNotice({ title, children }: SecurityNoticeProps) {
  return (
    <aside className="security-notice" role="note" aria-label={title}>
      <strong>{title}</strong>
      <p>{children}</p>
    </aside>
  );
}
