import React from "react";
import { useCurrentFrame, interpolate } from "remotion";
import type { ChartPoint } from "../data/dashboard";
import { theme } from "../theme";

export const LineChart: React.FC<{
  data: ChartPoint[];
  label: string;
  ySuffix?: string;
  color?: string;
}> = ({ data, label, ySuffix = "", color = theme.chart.line }) => {
  const frame = useCurrentFrame();
  const width = 860;
  const height = 260;
  const padding = { top: 25, right: 30, bottom: 35, left: 60 };
  const chartW = width - padding.left - padding.right;
  const chartH = height - padding.top - padding.bottom;

  const values = data.map((d) => d.value);
  const minVal = Math.min(...values) * 0.85;
  const maxVal = Math.max(...values) * 1.12;

  // Animate progress: draw line over time
  const drawProgress = interpolate(frame, [5, 70], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // Points
  const points = data.map((d, i) => {
    const x = padding.left + (i / (data.length - 1)) * chartW;
    const y = padding.top + chartH - ((d.value - minVal) / (maxVal - minVal)) * chartH;
    return { x, y, ...d };
  });

  const numVisiblePoints = Math.floor(drawProgress * data.length);
  const visiblePoints = points.slice(0, numVisiblePoints);

  const linePath = visiblePoints
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`)
    .join(" ");

  // Grid lines
  const gridLines = 5;
  const gridYs = Array.from({ length: gridLines }, (_, i) => {
    const y = padding.top + (i / (gridLines - 1)) * chartH;
    const val = maxVal - (i / (gridLines - 1)) * (maxVal - minVal);
    return { y, val };
  });

  // Label animations
  const labelOpacity = interpolate(frame, [0, 8], [0, 1], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  const gridOpacity = interpolate(frame, [5, 20], [0, 0.5], {
    extrapolateRight: "clamp",
    extrapolateLeft: "clamp",
  });

  // X labels (show every 3rd + last)
  const xLabels = points.filter((_, i) => i % 3 === 0 || i === data.length - 1);

  // Tooltip at the currently-animating point
  const tooltipIndex = Math.min(numVisiblePoints - 1, data.length - 1);
  const tooltipPoint = points[tooltipIndex >= 0 ? tooltipIndex : 0];
  const tooltipOpacity =
    numVisiblePoints > 1
      ? interpolate(frame, [70, 80], [0, 1], {
          extrapolateRight: "clamp",
          extrapolateLeft: "clamp",
        })
      : 0;

  return (
    <div style={{ position: "relative" }}>
      {/* Chart label */}
      <div
        style={{
          color: theme.text.secondary,
          fontSize: 13,
          fontFamily: theme.font.sans,
          fontWeight: 500,
          marginBottom: 8,
          opacity: labelOpacity,
          letterSpacing: "-0.01em",
        }}
      >
        {label}
      </div>

      <svg
        viewBox={`0 0 ${width} ${height}`}
        style={{ width: "100%", height: "auto" }}
      >
        {/* Grid lines */}
        {gridYs.map((g, i) => (
          <g key={i}>
            <line
              x1={padding.left}
              y1={g.y}
              x2={width - padding.right}
              y2={g.y}
              stroke={theme.chart.grid}
              strokeWidth={1}
              strokeDasharray="4 4"
              opacity={gridOpacity * (i === 0 || i === gridLines - 1 ? 0.7 : 0.4)}
            />
            <text
              x={padding.left - 8}
              y={g.y + 4}
              fill={theme.text.muted}
              fontSize={10}
              fontFamily={theme.font.sans}
              textAnchor="end"
              opacity={labelOpacity * 0.8}
            >
              {Math.round(g.val)}
              {ySuffix}
            </text>
          </g>
        ))}

        {/* X axis line */}
        <line
          x1={padding.left}
          y1={padding.top + chartH}
          x2={width - padding.right}
          y2={padding.top + chartH}
          stroke={theme.chart.grid}
          strokeWidth={1}
          opacity={labelOpacity * 0.6}
        />

        {/* X labels */}
        {xLabels.map((p, i) => (
          <text
            key={i}
            x={p.x}
            y={height - 8}
            fill={theme.text.muted}
            fontSize={10}
            fontFamily={theme.font.sans}
            textAnchor="middle"
            opacity={labelOpacity * 0.8}
          >
            {p.time}
          </text>
        ))}

        {/* Area fill under line */}
        {numVisiblePoints > 1 && (
          <path
            d={`${linePath} L ${visiblePoints[numVisiblePoints - 1].x} ${padding.top + chartH} L ${visiblePoints[0].x} ${padding.top + chartH} Z`}
            fill="url(#areaGradient)"
            opacity={0.4}
          />
        )}

        {/* Gradient definition */}
        <defs>
          <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.4} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
          <filter id="pointGlow">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        {/* Animated line */}
        <path
          d={linePath}
          fill="none"
          stroke={color}
          strokeWidth={2.5}
          strokeLinecap="round"
          strokeLinejoin="round"
          opacity={labelOpacity}
        />

        {/* Data points */}
        {visiblePoints.map((p, i) => {
          const pointOpacity =
            i === numVisiblePoints - 1
              ? 1
              : interpolate(i / numVisiblePoints, [0, 1], [0.2, 0.8]);
          const pointRadius = i === numVisiblePoints - 1 ? 5 : 3;

          return (
            <g key={i} opacity={pointOpacity * labelOpacity}>
              {/* Outer glow */}
              <circle
                cx={p.x}
                cy={p.y}
                r={pointRadius + 4}
                fill={color}
                opacity={i === numVisiblePoints - 1 ? 0.25 : 0.1}
              />
              {/* Inner dot */}
              <circle
                cx={p.x}
                cy={p.y}
                r={pointRadius}
                fill={color}
                filter={i === numVisiblePoints - 1 ? "url(#pointGlow)" : undefined}
              />
            </g>
          );
        })}

        {/* Tooltip */}
        {tooltipPoint && numVisiblePoints > 0 && (
          <g opacity={tooltipOpacity}>
            {/* Tooltip background */}
            <rect
              x={tooltipPoint.x - 40}
              y={tooltipPoint.y - 42}
              width={80}
              height={28}
              rx={6}
              fill="rgba(15, 15, 25, 0.9)"
              stroke={color}
              strokeWidth={1}
            />
            {/* Tooltip text */}
            <text
              x={tooltipPoint.x}
              y={tooltipPoint.y - 22}
              fill={theme.text.primary}
              fontSize={12}
              fontFamily={theme.font.sans}
              fontWeight={600}
              textAnchor="middle"
            >
              {tooltipPoint.value}
              {ySuffix}
            </text>
            {/* Tooltip connector line */}
            <line
              x1={tooltipPoint.x}
              y1={tooltipPoint.y - 14}
              x2={tooltipPoint.x}
              y2={tooltipPoint.y - 3}
              stroke={color}
              strokeWidth={1}
              strokeDasharray="2 2"
            />
          </g>
        )}
      </svg>
    </div>
  );
};
