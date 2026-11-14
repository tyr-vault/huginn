import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "./auth.jsx";
import LoginPage from "./pages/LoginPage.jsx";
import SessionsPage from "./pages/SessionsPage.jsx";
import NewSessionPage from "./pages/NewSessionPage.jsx";
import SessionDetailPage from "./pages/SessionDetailPage.jsx";

function PrivateRoute({ children }) {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return children;
}

function PublicHome() {
  const { isAuthenticated } = useAuth();
  if (isAuthenticated) return <Navigate to="/sessions" replace />;
  return <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/" element={<PublicHome />} />
      <Route
        path="/sessions"
        element={
          <PrivateRoute>
            <SessionsPage />
          </PrivateRoute>
        }
      />
      <Route
        path="/sessions/new"
        element={
          <PrivateRoute>
            <NewSessionPage />
          </PrivateRoute>
        }
      />
      <Route
        path="/sessions/:id"
        element={
          <PrivateRoute>
            <SessionDetailPage />
          </PrivateRoute>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
