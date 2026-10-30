export const BLEND_FRAMES_ANIMATION =
  "hoi4-blend-frames 1s linear infinite alternate";

export function BlendFramesKeyframes() {
  return (
    <style>
      {"@keyframes hoi4-blend-frames{from{opacity:0}to{opacity:1}}"}
    </style>
  );
}
