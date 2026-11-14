import React, { createContext, useContext, useMemo, useState } from "react";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem("huginn_token") || "");

  const value = useMemo(
    () => ({
      token,
      isAuthenticated: Boolean(token),
      login: (next) => {
        localStorage.setItem("huginn_token", next);
        setToken(next);
      },
      logout: () => {
        localStorage.removeItem("huginn_token");
        setToken("");
      },
    }),
    [token]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  return useContext(AuthContext);
}
