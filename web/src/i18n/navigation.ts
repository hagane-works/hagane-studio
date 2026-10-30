import { createElement, useMemo, type Ref } from "react";
import { useLocale } from "use-intl";
import { useLocation, useNavigate } from "react-router";
import PlatformLink, {
  hrefToString,
  type Href,
  type LinkProps,
} from "@/platform/link";
import { isAppLocale, LOCALE_STORAGE_KEY, type AppLocale } from "./routing";

export function getPathname({
  href,
  locale,
}: {
  href: Href;
  locale: AppLocale;
}): string {
  const path = hrefToString(href);
  if (
    /^[a-z][a-z0-9+.-]*:/i.test(path) ||
    path.startsWith("#") ||
    path.startsWith("//")
  )
    return path;
  const relative = path.startsWith("/") ? path : `/${path}`;
  return relative === "/" ? `/${locale}` : `/${locale}${relative}`;
}

export function stripLocale(pathname: string): string {
  const [, first, ...rest] = pathname.split("/");
  if (isAppLocale(first)) return `/${rest.join("/")}`;
  return pathname;
}

type LocaleLinkProps = LinkProps & {
  locale?: AppLocale;
  ref?: Ref<HTMLAnchorElement>;
};

export function Link({ href, locale, ...rest }: LocaleLinkProps) {
  const current = useLocale() as AppLocale;
  return createElement(PlatformLink, {
    href: getPathname({ href, locale: locale ?? current }),
    ...rest,
  });
}

export function usePathname(): string {
  return stripLocale(useLocation().pathname);
}

interface LocaleNavigateOptions {
  locale?: AppLocale;
  scroll?: boolean;
}

export function useRouter() {
  const navigate = useNavigate();
  const current = useLocale() as AppLocale;
  return useMemo(() => {
    const go = (
      href: Href,
      options: LocaleNavigateOptions | undefined,
      replace: boolean,
    ) => {
      const locale = options?.locale ?? current;
      if (options?.locale) {
        try {
          localStorage.setItem(LOCALE_STORAGE_KEY, options.locale);
        } catch {}
      }
      navigate(getPathname({ href, locale }), { replace });
      if (options?.scroll !== false && !replace) window.scrollTo(0, 0);
    };
    return {
      push: (href: Href, options?: LocaleNavigateOptions) =>
        go(href, options, false),
      replace: (href: Href, options?: LocaleNavigateOptions) =>
        go(href, options, true),
      back: () => navigate(-1),
      forward: () => navigate(1),
      refresh: () => {},
      prefetch: () => {},
    };
  }, [navigate, current]);
}

export class RedirectError extends Error {
  constructor(readonly href: string) {
    super(`Redirect to ${href}`);
    this.name = "RedirectError";
  }
}

export function redirect({
  href,
  locale,
}: {
  href: Href;
  locale: AppLocale;
}): never {
  throw new RedirectError(getPathname({ href, locale }));
}

export const permanentRedirect = redirect;
