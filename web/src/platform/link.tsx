import {
  forwardRef,
  type AnchorHTMLAttributes,
  type MouseEvent,
  type ReactNode,
} from "react";
import { Link as RouterLink } from "react-router";

export type Href =
  | string
  | {
      pathname?: string;
      query?: Record<string, string | number | boolean | undefined>;
      hash?: string;
    };

export interface LinkProps extends Omit<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  "href"
> {
  href: Href;
  replace?: boolean;

  prefetch?: boolean | null;
  scroll?: boolean;

  onNavigate?: (event: { preventDefault: () => void }) => void;
  children?: ReactNode;
}

export function hrefToString(href: Href): string {
  if (typeof href === "string") return href;
  let path = href.pathname ?? "";
  if (href.query) {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(href.query)) {
      if (value !== undefined) params.set(key, String(value));
    }
    const query = params.toString();
    if (query) path += `?${query}`;
  }
  if (href.hash)
    path += href.hash.startsWith("#") ? href.hash : `#${href.hash}`;
  return path;
}

function isExternal(path: string): boolean {
  return /^[a-z][a-z0-9+.-]*:/i.test(path) || path.startsWith("//");
}

const Link = forwardRef<HTMLAnchorElement, LinkProps>(function Link(
  {
    href,
    replace,
    prefetch: _prefetch,
    scroll: _scroll,
    onNavigate,
    onClick,
    target,
    download,
    ...rest
  },
  ref,
) {
  const path = hrefToString(href);
  if (
    isExternal(path) ||
    path.startsWith("#") ||
    download !== undefined ||
    (target && target !== "_self")
  ) {
    return (
      <a
        ref={ref}
        href={path}
        target={target}
        download={download}
        onClick={onClick}
        {...rest}
      />
    );
  }
  return (
    <RouterLink
      ref={ref}
      to={path}
      replace={replace}
      target={target}
      onClick={(event: MouseEvent<HTMLAnchorElement>) => {
        onClick?.(event);

        if (
          event.defaultPrevented ||
          !onNavigate ||
          event.button !== 0 ||
          event.metaKey ||
          event.ctrlKey ||
          event.shiftKey ||
          event.altKey
        )
          return;
        onNavigate({ preventDefault: () => event.preventDefault() });
      }}
      {...rest}
    />
  );
});

export default Link;

export function useLinkStatus(): { pending: boolean } {
  return { pending: false };
}
