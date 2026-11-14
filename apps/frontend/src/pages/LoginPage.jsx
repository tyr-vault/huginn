import { useState } from "react";
import { Navigate } from "react-router-dom";
import { api } from "../api.js";
import { useAuth } from "../auth.jsx";
import Brand from "../components/Brand.jsx";

export default function LoginPage() {
  const { isAuthenticated, login } = useAuth();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  if (isAuthenticated) return <Navigate to="/sessions" replace />;

  async function onSubmit(e) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const data = await api("/api/v1/auth/login", {
        method: "POST",
        body: { username, password },
      });
      login(data.token);
    } catch (err) {
      setError(err.message || "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="landing">
      <header className="landing-top">
        <Brand size="compact" />
      </header>

      <main className="landing-hero">
        <section className="landing-copy">
          <Brand size="landing" />
          <h1 className="landing-catchphrase">See the market. Capture the leads.</h1>
          <p className="landing-support">
            Self-hosted discovery from Google Maps into maps, CSV, and BI sessions.
          </p>
        </section>

        <form className="landing-form panel stack" onSubmit={onSubmit} aria-label="Sign in">
          <div>
            <h2 className="landing-form-title">Sign in</h2>
            <p className="muted">Use your admin credentials to open the control plane.</p>
          </div>
          <label>
            Username
            <input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" />
          </label>
          <label>
            Password
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
          </label>
          {error ? <div className="error">{error}</div> : null}
          <button type="submit" disabled={loading}>
            {loading ? "Signing in…" : "Get started"}
          </button>
        </form>
      </main>
    </div>
  );
}
