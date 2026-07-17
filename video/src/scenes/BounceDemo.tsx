import { TransitionSeries } from "@remotion/transitions";
import { fade } from "@remotion/transitions/fade";
import { slide } from "@remotion/transitions/slide";
import { Audio, staticFile } from "remotion";
import { Intro } from "./Intro";
import { Metrics } from "./Metrics";
import { Charts } from "./Charts";
import { ApiEndpoints } from "./ApiEndpoints";
import { Cta } from "./Cta";
import { sceneDurations } from "../data/dashboard";
import type { TransitionTiming } from "@remotion/transitions";

const fadeTiming: TransitionTiming = {
  getDurationInFrames: () => 15,
  getProgress: ({ frame }) => {
    return frame / 15;
  },
};

const slideTiming: TransitionTiming = {
  getDurationInFrames: () => 20,
  getProgress: ({ frame }) => {
    return frame / 20;
  },
};

export const BounceDemo: React.FC = () => {
  return (
    <>
      <Audio src={staticFile("ambient.wav")} volume={0.25} />
      <TransitionSeries>
        <TransitionSeries.Sequence durationInFrames={sceneDurations.intro}>
          <Intro />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={fadeTiming}
        />

        <TransitionSeries.Sequence durationInFrames={sceneDurations.metrics}>
          <Metrics />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={slide({ direction: "from-right" })}
          timing={slideTiming}
        />

        <TransitionSeries.Sequence durationInFrames={sceneDurations.charts}>
          <Charts />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={fade()}
          timing={fadeTiming}
        />

        <TransitionSeries.Sequence durationInFrames={sceneDurations.api}>
          <ApiEndpoints />
        </TransitionSeries.Sequence>

        <TransitionSeries.Transition
          presentation={slide({ direction: "from-right" })}
          timing={slideTiming}
        />

        <TransitionSeries.Sequence durationInFrames={sceneDurations.cta}>
          <Cta />
        </TransitionSeries.Sequence>
      </TransitionSeries>
    </>
  );
};
