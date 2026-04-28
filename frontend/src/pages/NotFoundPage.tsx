import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <main className="status-page">
      <div className="status-card">
        <p className="eyebrow">404</p>
        <h1>Page not found</h1>
        <p>The route exists in neither the lab flow nor the current demo shell.</p>
        <Link className="primary-link" to="/app">
          Back to workspace
        </Link>
      </div>
    </main>
  );
}
