import React from "react";
import { useCurrentFrame, interpolate, useVideoConfig, spring } from "remotion";
import type { MetricData } from "../data/dashboard";
import { theme } from "../theme";

export const MetricCard: React.FC<{
  metric: MetricData;
  index: number;
}> = ({ metric, index }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const delay = index * 8;

  // Card entrance spring
  const entrance = spring({
    frame: frame - delay,
    fps,
    config: { damping: 15, stiffness: 80 },
    durationInFrames: 20,
  });

  const scale = interpolate(entrance, [0, 1], [0.85, 1]);
  const opacity = interpolate(entrance, [0, 1], [0, 1]);

  // Counter animation
  const counterStart = delay + 10;
  const counterDuration = 40;
  const countProgress = interpolate(
    frame,
    [counterStart, counterStart + counterDuration],
    [0, 1],
    { extrapolateRight: "clamp", extrapolateLeft: "clamp" }
  );

  const displayValue =
    metric.value % 1 === 0
      ? Math.round(metric.value * countProgress)
      : (metric.value * countProgress).toFixed(1);

  const trendColor =
    metric.trend === "up"
      ? theme.chart.green
      : metric.trend === "down"
        ? "#ef4444"
        : theme.text.muted;

  const trendArrow = metric.trend === "up" ? "↑" : metric.trend === "down" ? "↓" : "→";

  return (
    <div
      style={{
        transform: `scale(${scale})`,
        opacity,
        background: theme.card.bg,
        border: `1px solid ${theme.card.border}`,
        borderRadius: 16,
        padding: "28px 32px",
        display: "flex",
        flexDirection: "column",
        gap: 12,
        backdropFilter: "blur(12px)",
        WebkitBackdropFilter: "blur(12px)",
        position: "relative",
        overflow: "hidden",
      }}
    >
      {/* Glow effect on hover-like pulse */}
      <div
        style={{
          position: "absolute",
          top: 0,
          left: 0,
          right: 0,
          height: 1,
          background: `linear-gradient(90deg, transparent, ${theme.accentGlow}, transparent)`,
          opacity: interpolate(entrance, [0, 1], [0, 0.6]),
        }}
      />

      {/* Header row */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
        }}
      >
        <span
          style={{
            color: theme.text.secondary,
            fontSize: 14,
            fontFamily: theme.font.sans,
            fontWeight: 500,
            letterSpacing: "-0.01em",
          }}
        >
          {metric.label}
        </span>
        <span style={{ fontSize: 20 }}>{metric.icon}</span>
      </div>

      {/* Value */}
      <div
        style={{
          display: "flex",
          alignItems: "baseline",
          gap: 4,
        }}
      >
        <span
          style={{
            color: theme.text.primary,
            fontSize: 40,
            fontFamily: theme.font.sans,
            fontWeight: 700,
            letterSpacing: "-0.02em",
            lineHeight: 1,
          }}
        >
          {displayValue}
        </span>
        <span
          style={{
            color: theme.text.secondary,
            fontSize: 18,
            fontFamily: theme.font.sans,
            fontWeight: 500,
          }}
        >
          {metric.suffix}
        </span>
      </div>

      {/* Trend */}
      <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
        <span style={{ color: trendColor, fontSize: 14, fontWeight: 600 }}>
          {trendArrow}
        </span>
        <span
          style={{
            color: theme.text.muted,
            fontSize: 12,
            fontFamily: theme.font.sans,
          }}
        >
          vs última hora
        </span>
      </div>
    </div>
  );
};
