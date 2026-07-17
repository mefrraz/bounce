export const theme = {
  background: {
    from: "#0a0a0f",
    to: "#1a1a2e",
  },
  accent: "#F97316",
  accentLight: "#FB923C",
  accentGlow: "rgba(249, 115, 22, 0.3)",
  text: {
    primary: "#ffffff",
    secondary: "rgba(255, 255, 255, 0.7)",
    muted: "rgba(255, 255, 255, 0.45)",
  },
  card: {
    bg: "rgba(255, 255, 255, 0.04)",
    border: "rgba(255, 255, 255, 0.08)",
    hover: "rgba(255, 255, 255, 0.06)",
  },
  chart: {
    line: "#F97316",
    area: "rgba(249, 115, 22, 0.15)",
    grid: "rgba(255, 255, 255, 0.06)",
    green: "#22c55e",
  },
  font: {
    sans: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    mono: '"JetBrains Mono", "Fira Code", monospace',
  },
} as const;
