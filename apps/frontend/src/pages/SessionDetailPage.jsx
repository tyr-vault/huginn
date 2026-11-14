import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { MapContainer, Marker, Popup, TileLayer } from "react-leaflet";
import L from "leaflet";
import markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import markerIcon from "leaflet/dist/images/marker-icon.png";
import markerShadow from "leaflet/dist/images/marker-shadow.png";
import { api } from "../api.js";
import { useAuth } from "../auth.jsx";
import Brand from "../components/Brand.jsx";

delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

export default function SessionDetailPage() {
  const { id } = useParams();
  const { token } = useAuth();
  const navigate = useNavigate();
  const [session, setSession] = useState(null);
  const [leads, setLeads] = useState([]);
  const [error, setError] = useState("");

  async function load() {
    try {
      const [sess, leadRows] = await Promise.all([
        api(`/api/v1/sessions/${id}`, { token }),
        api(`/api/v1/sessions/${id}/leads`, { token }),
      ]);
      setSession(sess);
      setLeads(leadRows || []);
      setError("");
    } catch (err) {
      setError(err.message);
    }
  }

  useEffect(() => {
    load();
    const timer = setInterval(load, 3000);
    return () => clearInterval(timer);
  }, [id, token]);

  const points = useMemo(
    () => leads.filter((l) => Number(l.latitude) && Number(l.longitude)),
    [leads]
  );

  const center = points.length
    ? [points[0].latitude, points[0].longitude]
    : [-23.18, -45.88];

  async function erase() {
    if (!confirm("Delete this session and all leads?")) return;
    await api(`/api/v1/sessions/${id}`, { token, method: "DELETE" });
    navigate("/");
  }

  async function exportCsv() {
    const res = await fetch(`/api/v1/sessions/${id}/export.csv`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) {
      setError("Export failed");
      return;
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `session-${id}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  if (!session) {
    return (
      <div className="shell">
        <div className="panel">{error || "Loading…"}</div>
      </div>
    );
  }

  return (
    <div className="shell stack">
      <div className="topbar">
        <Brand size="compact" linkHome />
        <div className="row">
          <Link className="btn secondary" to="/sessions">
            Sessions
          </Link>
          <button type="button" onClick={exportCsv}>
            Export CSV
          </button>
          <button type="button" className="danger" onClick={erase}>
            Erase session
          </button>
        </div>
      </div>

      <div className="stack" style={{ gap: "0.35rem", marginBottom: "0.5rem" }}>
        <h1 className="page-title">{session.name}</h1>
        <div className="row">
          <span className={`badge ${session.status}`}>{session.status}</span>
          <span className="muted">{session.lead_count ?? leads.length} leads</span>
        </div>
      </div>

      {error ? <div className="error">{error}</div> : null}
      {session.error ? <div className="error">{session.error}</div> : null}

      <div className="grid-2">
        <div className="panel stack">
          <h2 style={{ margin: 0, fontFamily: "var(--display)" }}>Map</h2>
          <div className="map-wrap">
            <MapContainer center={center} zoom={12} style={{ height: "100%", width: "100%" }}>
              <TileLayer
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a>'
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              />
              {points.map((p, idx) => (
                <Marker key={`${p.place_id}-${idx}`} position={[p.latitude, p.longitude]}>
                  <Popup>
                    <strong>{p.name}</strong>
                    <br />
                    {p.address || p.category}
                  </Popup>
                </Marker>
              ))}
            </MapContainer>
          </div>
        </div>

        <div className="panel stack">
          <h2 style={{ margin: 0, fontFamily: "var(--display)" }}>Leads</h2>
          <div style={{ maxHeight: 420, overflow: "auto" }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Phone</th>
                  <th>City</th>
                  <th>Rating</th>
                </tr>
              </thead>
              <tbody>
                {leads.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="muted">
                      {session.status === "running" || session.status === "pending"
                        ? "Extraction in progress…"
                        : "No leads yet."}
                    </td>
                  </tr>
                ) : (
                  leads.map((l, idx) => (
                    <tr key={`${l.place_id}-${idx}`}>
                      <td>
                        {l.maps_url ? (
                          <a href={l.maps_url} target="_blank" rel="noreferrer">
                            {l.name}
                          </a>
                        ) : (
                          l.name
                        )}
                      </td>
                      <td>{l.phone}</td>
                      <td>{l.city}</td>
                      <td>{l.rating || ""}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
