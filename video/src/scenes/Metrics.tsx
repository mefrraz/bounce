import React from "react";
import {
  useCurrentFrame,
  interpolate,
  AbsoluteFill,
} from "remotion";
import { theme } from "../theme";
import { metrics } from "../data/dashboard";
import { MetricCard } from "../components/MetricCard";

export const Metrics: React.FC = () => {
  const frame = useCurrentFrame();

  // Section title
  const titleOpacity = interpolate(frame, [0, 10], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // Live indicator pulse
  const livePulse = interpolate(
    Math.sin((frame / 15) * Math.PI * 2),
    [-1, 1],
    [0.4, 1]
  );

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
          marginBottom: 40,
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
          📊 Métricas em Tempo Real
        </h2>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
          }}
        >
          <div
            style={{
              width: 8,
              height: 8,
              borderRadius: "50%",
              background: theme.chart.green,
              opacity: livePulse,
              boxShadow: `0 0 8px ${theme.chart.green}`,
            }}
          />
          <span
            style={{
              color: theme.text.muted,
              fontSize: 13,
              fontFamily: theme.font.mono,
            }}
          >
            LIVE
          </span>
        </div>
      </div>

      {/* Metric cards grid */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: 24,
          flex: 1,
        }}
      >
        {metrics.map((metric, i) => (
          <MetricCard key={metric.label} metric={metric} index={i} />
        ))}
      </div>
    </AbsoluteFill>
  );
};
