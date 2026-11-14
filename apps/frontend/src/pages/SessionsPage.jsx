import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api.js";
import { useAuth } from "../auth.jsx";
import Brand from "../components/Brand.jsx";

export default function SessionsPage() {
  const { token, logout } = useAuth();
  const [sessions, setSessions] = useState([]);
  const [error, setError] = useState("");

  async function load() {
    try {
      const data = await api("/api/v1/sessions", { token });
      setSessions(data || []);
      setError("");
    } catch (err) {
      setError(err.message);
    }
  }

  useEffect(() => {
    load();
    const id = setInterval(load, 4000);
    return () => clearInterval(id);
  }, [token]);

  async function erase(id) {
    if (!confirm("Delete this session and all leads?")) return;
    try {
      await api(`/api/v1/sessions/${id}`, { token, method: "DELETE" });
      await load();
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div className="shell">
      <div className="topbar">
        <Brand size="compact" linkHome />
        <div className="row">
          <Link className="btn" to="/sessions/new">
            New session
          </Link>
          <button type="button" className="secondary" onClick={logout}>
            Log out
          </button>
        </div>
      </div>

      <div className="panel stack">
        <div>
          <h1 style={{ margin: 0, fontFamily: "var(--display)", fontWeight: 600 }}>BI sessions</h1>
          <p className="muted">Each session is one extraction operation with its own lead set.</p>
        </div>
        {error ? <div className="error">{error}</div> : null}
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Leads</th>
              <th>Created</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {sessions.length === 0 ? (
              <tr>
                <td colSpan={5} className="muted">
                  No sessions yet.
                </td>
              </tr>
            ) : (
              sessions.map((s) => (
                <tr key={s.id}>
                  <td>
                    <Link to={`/sessions/${s.id}`}>{s.name}</Link>
                  </td>
                  <td>
                    <span className={`badge ${s.status}`}>{s.status}</span>
                  </td>
                  <td>{s.lead_count ?? 0}</td>
                  <td>{new Date(s.created_at).toLocaleString()}</td>
                  <td>
                    <button type="button" className="danger" onClick={() => erase(s.id)}>
                      Erase
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
