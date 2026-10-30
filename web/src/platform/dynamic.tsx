import {
  lazy,
  Suspense,
  type ComponentType,
  type JSX,
  type ReactNode,
} from "react";

type Loaded<P> = ComponentType<P> | { default: ComponentType<P> };

interface DynamicOptions {
  ssr?: boolean;
  loading?: () => ReactNode;
}

export default function dynamic<P extends object>(
  load: () => Promise<Loaded<P>>,
  options: DynamicOptions = {},
): ComponentType<P> {
  const Lazy = lazy(async () => {
    const loaded = await load();
    return { default: "default" in loaded ? loaded.default : loaded };
  });
  function DynamicComponent(props: P) {
    return (
      <Suspense fallback={options.loading ? options.loading() : null}>
        <Lazy {...(props as P & JSX.IntrinsicAttributes)} />
      </Suspense>
    );
  }
  return DynamicComponent;
}
