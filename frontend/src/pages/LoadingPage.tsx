export function LoadingPage({ label = "Loading" }: { label?: string }) {
  return (
    <main className="status-page status-loading">
      <div className="status-card">
        <p className="eyebrow">Please wait</p>
        <h1>{label}</h1>
        <p>We are preparing the next screen and syncing the latest auth state.</p>
      </div>
    </main>
  );
}
