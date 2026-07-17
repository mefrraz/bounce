import React from "react";
import {
  useCurrentFrame,
  interpolate,
  useVideoConfig,
  spring,
  AbsoluteFill,
} from "remotion";
import { theme } from "../theme";

export const Cta: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  // Title entrance
  const titleS = spring({
    frame,
    fps,
    config: { damping: 12, stiffness: 50 },
    durationInFrames: 20,
  });

  // URL appearance
  const urlOpacity = interpolate(frame, [30, 45], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // Docker command
  const dockerOpacity = interpolate(frame, [55, 70], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // Glow pulse
  const glowPulse = interpolate(
    Math.sin((frame / 25) * Math.PI * 2),
    [-1, 1],
    [0.5, 1]
  );

  return (
    <AbsoluteFill
      style={{
        background: `linear-gradient(135deg, ${theme.background.from}, ${theme.background.to})`,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        padding: "0 80px",
      }}
    >
      {/* Background glow */}
      <div
        style={{
          position: "absolute",
          width: 500,
          height: 500,
          borderRadius: "50%",
          background: `radial-gradient(circle, ${theme.accentGlow}, transparent 60%)`,
          opacity: glowPulse * 0.4,
        }}
      />

      {/* Main CTA */}
      <h2
        style={{
          color: theme.text.primary,
          fontSize: 52,
          fontFamily: theme.font.sans,
          fontWeight: 800,
          letterSpacing: "-0.03em",
          margin: 0,
          transform: `scale(${titleS})`,
          textAlign: "center",
        }}
      >
        Experimenta a API
      </h2>

      {/* Decorative line */}
      <div
        style={{
          width: interpolate(frame, [10, 30], [0, 200], {
            extrapolateRight: "clamp",
            extrapolateLeft: "clamp",
          }),
          height: 2,
          background: `linear-gradient(90deg, transparent, ${theme.accent}, transparent)`,
          marginTop: 20,
          marginBottom: 30,
        }}
      />

      {/* GitHub URL */}
      <div
        style={{
          opacity: urlOpacity,
          display: "flex",
          alignItems: "center",
          gap: 10,
          padding: "14px 28px",
          borderRadius: 12,
          background: theme.card.bg,
          border: `1px solid ${theme.card.border}`,
          marginBottom: 24,
        }}
      >
        <span style={{ fontSize: 22 }}>🔗</span>
        <span
          style={{
            color: theme.text.primary,
            fontSize: 20,
            fontFamily: theme.font.mono,
            fontWeight: 500,
            letterSpacing: "-0.01em",
          }}
        >
          github.com/mefrraz/bounce
        </span>
      </div>

      {/* Docker command */}
      <div
        style={{
          opacity: dockerOpacity,
          padding: "16px 28px",
          borderRadius: 10,
          background: "rgba(0,0,0,0.4)",
          border: `1px solid ${theme.card.border}`,
        }}
      >
        <span
          style={{
            color: theme.accentLight,
            fontSize: 15,
            fontFamily: theme.font.mono,
            fontWeight: 400,
          }}
        >
          $ docker run -d -p 3001:3001 ghcr.io/mefrraz/bounce
        </span>
      </div>
    </AbsoluteFill>
  );
};
