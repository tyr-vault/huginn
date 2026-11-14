import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api.js";
import { useAuth } from "../auth.jsx";

export default function NewSessionPage() {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({
    name: "",
    query: "",
    location: "",
    max_results: 100,
    extraction_depth: 10,
    lang: "en",
  });

  function setField(key, value) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  async function onSubmit(e) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const session = await api("/api/v1/sessions", {
        token,
        method: "POST",
        body: {
          name: form.name || `${form.query} — ${form.location}`.trim(),
          config: {
            query: form.query,
            location: form.location,
            max_results: Number(form.max_results) || 100,
            extraction_depth: Number(form.extraction_depth) || 10,
            lang: form.lang || "en",
          },
        },
      });
      navigate(`/sessions/${session.id}`);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="shell">
      <div className="topbar">
        <div className="brand">Huginn</div>
        <Link className="btn secondary" to="/">
          Back
        </Link>
      </div>
      <form className="panel stack" onSubmit={onSubmit}>
        <div>
          <h1 style={{ margin: 0, fontFamily: "var(--display)" }}>Configure extraction</h1>
          <p className="muted">Start a BI session. Leads are stored in Postgres when the scrape finishes.</p>
        </div>
        <label>
          Session name
          <input value={form.name} onChange={(e) => setField("name", e.target.value)} placeholder="Optional" />
        </label>
        <label>
          Query
          <input required value={form.query} onChange={(e) => setField("query", e.target.value)} placeholder="barbershops" />
        </label>
        <label>
          Location
          <input value={form.location} onChange={(e) => setField("location", e.target.value)} placeholder="São José dos Campos, SP" />
        </label>
        <div className="grid-2">
          <label>
            Max results
            <input type="number" min="1" value={form.max_results} onChange={(e) => setField("max_results", e.target.value)} />
          </label>
          <label>
            Extraction depth
            <input
              type="number"
              min="1"
              value={form.extraction_depth}
              onChange={(e) => setField("extraction_depth", e.target.value)}
            />
          </label>
        </div>
        <label>
          Language
          <input value={form.lang} onChange={(e) => setField("lang", e.target.value)} />
        </label>
        {error ? <div className="error">{error}</div> : null}
        <button type="submit" disabled={loading}>
          {loading ? "Starting…" : "Run extraction"}
        </button>
      </form>
    </div>
  );
}
