import { useMemo } from "react";
import {
  useLocation,
  useNavigate,
  useParams as useRouteParams,
} from "react-router";

type Params = Record<string, string | string[]>;

export function useParams<T extends Params = Params>(): T {
  return useRouteParams() as unknown as T;
}

export function usePathname(): string {
  return useLocation().pathname;
}

export function useSearchParams(): URLSearchParams {
  const { search } = useLocation();
  return useMemo(() => new URLSearchParams(search), [search]);
}

export interface NavigateOptions {
  scroll?: boolean;
}

export interface AppRouter {
  push(href: string, options?: NavigateOptions): void;
  replace(href: string, options?: NavigateOptions): void;
  back(): void;
  forward(): void;

  refresh(): void;
  prefetch(href: string): void;
}

export function useRouter(): AppRouter {
  const navigate = useNavigate();
  return useMemo<AppRouter>(
    () => ({
      push: (href, options) => {
        navigate(href);
        if (options?.scroll !== false) window.scrollTo(0, 0);
      },
      replace: (href) => navigate(href, { replace: true }),
      back: () => navigate(-1),
      forward: () => navigate(1),
      refresh: () => {},
      prefetch: () => {},
    }),
    [navigate],
  );
}

export class NotFoundError extends Error {
  constructor() {
    super("Not found");
    this.name = "NotFoundError";
  }
}

export function notFound(): never {
  throw new NotFoundError();
}
