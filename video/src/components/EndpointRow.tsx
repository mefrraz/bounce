import React from "react";
import { useCurrentFrame, interpolate, spring } from "remotion";
import type { ApiEndpoint as ApiEndpointData } from "../data/dashboard";
import { theme } from "../theme";

export const EndpointRow: React.FC<{
  endpoint: ApiEndpointData;
  index: number;
}> = ({ endpoint, index }) => {
  const frame = useCurrentFrame();
  const delay = index * 14;
  const entrance = spring({
    frame: frame - delay,
    fps: 30,
    config: { damping: 18, stiffness: 90 },
    durationInFrames: 20,
  });

  const translateX = interpolate(entrance, [0, 1], [80, 0]);
  const opacity = interpolate(entrance, [0, 1], [0, 1]);

  const methodColor = endpoint.method === "WS" ? theme.chart.green : theme.accent;

  return (
    <div
      style={{
        transform: `translateX(${translateX}px)`,
        opacity,
        display: "flex",
        alignItems: "center",
        gap: 16,
        padding: "14px 20px",
        borderRadius: 10,
        background: theme.card.bg,
        border: `1px solid ${theme.card.border}`,
        marginBottom: 10,
      }}
    >
      {/* Method badge */}
      <span
        style={{
          display: "inline-block",
          padding: "3px 10px",
          borderRadius: 6,
          background: `${methodColor}20`,
          color: methodColor,
          fontSize: 12,
          fontFamily: theme.font.mono,
          fontWeight: 600,
          letterSpacing: "0.02em",
          minWidth: 42,
          textAlign: "center",
        }}
      >
        {endpoint.method}
      </span>

      {/* Path */}
      <span
        style={{
          color: theme.text.primary,
          fontSize: 15,
          fontFamily: theme.font.mono,
          fontWeight: 500,
          letterSpacing: "-0.01em",
        }}
      >
        {endpoint.path}
      </span>

      {/* Description */}
      <span
        style={{
          color: theme.text.muted,
          fontSize: 13,
          fontFamily: theme.font.sans,
          marginLeft: "auto",
        }}
      >
        {endpoint.description}
      </span>
    </div>
  );
};
