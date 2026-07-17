import React from "react";
import {
  useCurrentFrame,
  interpolate,
  useVideoConfig,
  spring,
  AbsoluteFill,
} from "remotion";
import { theme } from "../theme";

export const Intro: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  // Logo entrance
  const logoS = spring({
    frame,
    fps,
    config: { damping: 12, stiffness: 60 },
    durationInFrames: 25,
  });

  // Title fade
  const titleOpacity = interpolate(frame, [15, 30], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // Subtitle
  const subOpacity = interpolate(frame, [30, 45], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // Glow pulse
  const glowScale = interpolate(
    Math.sin((frame / 30) * Math.PI * 2),
    [-1, 1],
    [0.95, 1.08]
  );

  return (
    <AbsoluteFill
      style={{
        background: `linear-gradient(135deg, ${theme.background.from}, ${theme.background.to})`,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      {/* Background glow */}
      <div
        style={{
          position: "absolute",
          width: 300,
          height: 300,
          borderRadius: "50%",
          background: `radial-gradient(circle, ${theme.accentGlow}, transparent 70%)`,
          transform: `scale(${glowScale})`,
          opacity: interpolate(logoS, [0, 1], [0, 0.6]),
        }}
      />

      {/* Logo circle */}
      <div
        style={{
          width: 100,
          height: 100,
          borderRadius: "50%",
          background: `linear-gradient(135deg, ${theme.accent}, ${theme.accentLight})`,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          transform: `scale(${logoS})`,
          boxShadow: `0 0 60px ${theme.accentGlow}`,
        }}
      >
        <span style={{ fontSize: 48 }}>🏀</span>
      </div>

      {/* Title */}
      <h1
        style={{
          color: theme.text.primary,
          fontSize: 64,
          fontFamily: theme.font.sans,
          fontWeight: 800,
          letterSpacing: "-0.03em",
          marginTop: 28,
          marginBottom: 0,
          opacity: titleOpacity,
        }}
      >
        Bounce
      </h1>

      {/* Subtitle */}
      <p
        style={{
          color: theme.text.secondary,
          fontSize: 22,
          fontFamily: theme.font.sans,
          fontWeight: 400,
          letterSpacing: "0.04em",
          marginTop: 10,
          opacity: subOpacity,
        }}
      >
        Smart Sports Data Proxy
      </p>

      {/* Decorative line */}
      <div
        style={{
          width: interpolate(frame, [30, 50], [0, 120], {
            extrapolateRight: "clamp",
            extrapolateLeft: "clamp",
          }),
          height: 2,
          background: `linear-gradient(90deg, transparent, ${theme.accent}, transparent)`,
          marginTop: 20,
        }}
      />
    </AbsoluteFill>
  );
};
