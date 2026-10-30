import {
  Component,
  Suspense,
  lazy,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import {
  BrowserRouter,
  Navigate,
  Outlet,
  Route,
  Routes,
  useLocation,
  useParams,
} from "react-router";
import { Toaster } from "sonner";
import { SWRProvider } from "@/components/SWRProvider";
import { I18nProvider } from "@/components/i18n/I18nProvider";
import { ConfirmDialogProvider } from "@/components/shared/ConfirmDialogProvider";
import { BackendNotifier } from "@/components/BackendNotifier";
import { BackgroundImportWidget } from "@/components/workbench/BackgroundImportWidget";
import { ErrorBoundaryFallback } from "@/components/error/ErrorBoundaryFallback";
import ProjectSectionLoading from "@/app/[locale]/project/[id]/loading";
import { isAppLocale, preferredLocale, type AppLocale } from "@/i18n/routing";
import { loadMessages } from "@/i18n/messages";
import { RedirectError, stripLocale } from "@/i18n/navigation";
import { NotFoundError } from "@/platform/navigation";
import { ProgramWindowClose } from "./programWindow";
import { SetupGate } from "./SetupGate";
import { projectPages } from "./routes";

const HomePage = lazy(() => import("@/app/[locale]/page"));
const ProjectLayout = lazy(() => import("@/app/[locale]/project/[id]/layout"));
const GameViewPage = lazy(() => import("@/app/[locale]/game-view/page"));

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/:locale" element={<LocaleRoot />}>
          <Route
            index
            element={
              <Page>
                <HomePage />
              </Page>
            }
          />
          <Route path="project/:id" element={<ProjectShell />}>
            <Route index element={<Navigate to="start" replace />} />
            {Object.entries(projectPages).map(([path, PageComponent]) => (
              <Route key={path} path={path} element={<PageComponent />} />
            ))}
          </Route>
          <Route
            path="game-view"
            element={
              <Page>
                <GameViewPage />
              </Page>
            }
          />
          <Route path="*" element={<NotFound />} />
        </Route>
        <Route path="*" element={<LocaleRedirect />} />
      </Routes>
    </BrowserRouter>
  );
}

function LocaleRedirect() {
  const { pathname, search, hash } = useLocation();
  return (
    <Navigate
      to={`/${preferredLocale()}${stripLocale(pathname) === "/" ? "" : stripLocale(pathname)}${search}${hash}`}
      replace
    />
  );
}

function LocaleRoot() {
  const { locale } = useParams();
  const [messages, setMessages] = useState<{
    locale: AppLocale;
    data: Record<string, unknown>;
  } | null>(null);

  useEffect(() => {
    if (!isAppLocale(locale)) return;
    let cancelled = false;
    loadMessages(locale).then((data) => {
      if (!cancelled) setMessages({ locale, data });
    });
    return () => {
      cancelled = true;
    };
  }, [locale]);

  if (!isAppLocale(locale)) return <LocaleRedirect />;
  if (!messages) return null;

  return (
    <SWRProvider>
      <I18nProvider locale={messages.locale} messages={messages.data}>
        <ConfirmDialogProvider>
          <BackendNotifier />
          <ProgramWindowClose />
          <SetupGate>
            <Outlet />
          </SetupGate>
          <BackgroundImportWidget />
        </ConfirmDialogProvider>
      </I18nProvider>
      <Toaster
        position="top-right"
        richColors
        closeButton
        toastOptions={{ classNames: { toast: "ui-interactive" } }}
      />
    </SWRProvider>
  );
}

function Page({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  return (
    <RouteErrorBoundary key={pathname} scope="root">
      <Suspense fallback={null}>{children}</Suspense>
    </RouteErrorBoundary>
  );
}

function ProjectShell() {
  const { pathname } = useLocation();
  return (
    <Suspense fallback={<ProjectSectionLoading />}>
      <ProjectLayout>
        <RouteErrorBoundary key={pathname} scope="workbench">
          <Suspense fallback={<ProjectSectionLoading />}>
            <Outlet />
          </Suspense>
        </RouteErrorBoundary>
      </ProjectLayout>
    </Suspense>
  );
}

function NotFound() {
  return <Navigate to=".." replace />;
}

interface BoundaryProps {
  scope: "root" | "workbench";
  children: ReactNode;
}

class RouteErrorBoundary extends Component<
  BoundaryProps,
  { error: Error | null }
> {
  state = { error: null as Error | null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error) {
    if (
      !(error instanceof NotFoundError) &&
      !(error instanceof RedirectError)
    ) {
      console.error(error);
    }
  }

  render() {
    const { error } = this.state;
    if (!error) return this.props.children;
    if (error instanceof RedirectError)
      return <Navigate to={error.href} replace />;
    if (error instanceof NotFoundError) return <NotFound />;
    return (
      <ErrorBoundaryFallback
        error={error}
        reset={() => this.setState({ error: null })}
        scope={this.props.scope}
      />
    );
  }
}
