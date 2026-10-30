import { forwardRef, type CSSProperties, type ImgHTMLAttributes } from "react";

export interface StaticImageData {
  src: string;
  width?: number;
  height?: number;
}

export interface ImageProps extends Omit<
  ImgHTMLAttributes<HTMLImageElement>,
  "src" | "width" | "height"
> {
  src: string | StaticImageData;
  width?: number | `${number}`;
  height?: number | `${number}`;
  fill?: boolean;

  priority?: boolean;
  unoptimized?: boolean;
  quality?: number;
  placeholder?: string;
  blurDataURL?: string;
  loader?: unknown;
  overrideSrc?: string;
}

const Image = forwardRef<HTMLImageElement, ImageProps>(function Image(
  {
    src,
    width,
    height,
    fill,
    style,
    priority,
    unoptimized: _unoptimized,
    quality: _quality,
    placeholder: _placeholder,
    blurDataURL: _blurDataURL,
    loader: _loader,
    overrideSrc,
    loading,
    decoding,
    ...rest
  },
  ref,
) {
  const url = overrideSrc ?? (typeof src === "string" ? src : src.src);
  const fillStyle: CSSProperties | undefined = fill
    ? { position: "absolute", inset: 0, width: "100%", height: "100%" }
    : undefined;
  return (
    <img
      ref={ref}
      src={url}
      width={
        fill
          ? undefined
          : (width ?? (typeof src === "object" ? src.width : undefined))
      }
      height={
        fill
          ? undefined
          : (height ?? (typeof src === "object" ? src.height : undefined))
      }
      loading={loading ?? (priority ? "eager" : "lazy")}
      decoding={decoding ?? "async"}
      style={fillStyle ? { ...fillStyle, ...style } : style}
      {...rest}
    />
  );
});

export default Image;
