import React from "react";
import {
  useCurrentFrame,
  interpolate,
  AbsoluteFill,
} from "remotion";
import { theme } from "../theme";
import { endpoints } from "../data/dashboard";
import { EndpointRow } from "../components/EndpointRow";

export const ApiEndpoints: React.FC = () => {
  const frame = useCurrentFrame();

  // Section title
  const titleOpacity = interpolate(frame, [0, 10], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // Tagline
  const taglineOpacity = interpolate(frame, [5, 15], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

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
          alignItems: "baseline",
          gap: 20,
          marginBottom: 32,
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
          🔌 API REST
        </h2>
        <span
          style={{
            color: theme.text.muted,
            fontSize: 14,
            fontFamily: theme.font.sans,
            opacity: taglineOpacity,
          }}
        >
          REST · JSON · WebSocket
        </span>
      </div>

      {/* Endpoint list */}
      <div style={{ flex: 1 }}>
        {endpoints.map((ep, i) => (
          <EndpointRow key={ep.path} endpoint={ep} index={i} />
        ))}
      </div>
    </AbsoluteFill>
  );
};
