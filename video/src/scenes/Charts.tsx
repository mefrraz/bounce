import React from "react";
import {
  useCurrentFrame,
  interpolate,
  AbsoluteFill,
} from "remotion";
import { theme } from "../theme";
import { chartData, cacheHitData } from "../data/dashboard";
import { LineChart } from "../components/LineChart";

export const Charts: React.FC = () => {
  const frame = useCurrentFrame();

  // Section title
  const titleOpacity = interpolate(frame, [0, 10], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // Time selector highlight
  const selectorOpacity = interpolate(frame, [20, 30], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  const timeOptions = ["1m", "5m", "1h", "6h", "24h", "7d"];

  return (
    <AbsoluteFill
      style={{
        background: `linear-gradient(135deg, ${theme.background.from}, ${theme.background.to})`,
        display: "flex",
        flexDirection: "column",
        padding: "50px 80px",
      }}
    >
      {/* Header */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 30,
          opacity: titleOpacity,
        }}
      >
        <h2
          style={{
            color: theme.text.primary,
            fontSize: 28,
            fontFamily: theme.font.sans,
            fontWeight: 600,
            letterSpacing: "-0.02em",
            margin: 0,
          }}
        >
          📈 Gráficos de Performance
        </h2>

        {/* Time selector pills */}
        <div
          style={{
            display: "flex",
            gap: 6,
            opacity: selectorOpacity,
          }}
        >
          {timeOptions.map((t, i) => (
            <span
              key={t}
              style={{
                padding: "4px 12px",
                borderRadius: 20,
                fontSize: 12,
                fontFamily: theme.font.sans,
                fontWeight: 500,
                background:
                  i === 2 ? theme.accent : "rgba(255,255,255,0.06)",
                color: i === 2 ? "#fff" : theme.text.muted,
              }}
            >
              {t}
            </span>
          ))}
        </div>
      </div>

      {/* Charts */}
      <div style={{ display: "flex", flexDirection: "column", gap: 30, flex: 1 }}>
        <LineChart
          data={chartData}
          label="Requests / minuto"
          color={theme.chart.line}
        />
        <LineChart
          data={cacheHitData}
          label="Cache Hit Rate"
          ySuffix="%"
          color={theme.chart.green}
        />
      </div>
    </AbsoluteFill>
  );
};
