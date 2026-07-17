import { Composition } from "remotion";
import { BounceDemo } from "./scenes/BounceDemo";
import { totalFrames } from "./data/dashboard";

export const BounceComposition = () => {
  return (
    <Composition
      id="BounceDemo"
      component={BounceDemo}
      durationInFrames={totalFrames}
      fps={30}
      width={1920}
      height={1080}
    />
  );
};
