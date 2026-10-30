"use client";

import { Component, type ErrorInfo, type ReactNode, useMemo } from "react";
import { EyeOff, RotateCcw } from "lucide-react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";

interface BoundaryCopy {
  title: string;
  body: string;
  retry: string;
}

interface InnerProps {
  children: ReactNode;
  copy: BoundaryCopy;
  resetKey: string | number;
}

interface InnerState {
  error: Error | null;
}

class EventPreviewBoundaryInner extends Component<InnerProps, InnerState> {
  state: InnerState = { error: null };

  static getDerivedStateFromError(error: Error): InnerState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("[event-preview]", error, info.componentStack);
  }

  componentDidUpdate(previous: InnerProps) {
    if (previous.resetKey !== this.props.resetKey && this.state.error) {
      this.setState({ error: null });
    }
  }

  retry = () => this.setState({ error: null });

  render() {
    if (!this.state.error) return this.props.children;

    return (
      <div className="flex h-full min-h-48 items-center justify-center p-4">
        <div className="w-full rounded-lg border border-border bg-muted/20 p-4 text-center">
          <span
            className="mx-auto flex h-9 w-9 items-center justify-center rounded-md bg-muted"
            aria-hidden
          >
            <EyeOff className="h-5 w-5 text-muted-foreground" />
          </span>
          <p className="mt-3 text-sm font-medium text-foreground">
            {this.props.copy.title}
          </p>
          <p className="mt-1 text-xs leading-normal text-muted-foreground">
            {this.props.copy.body}
          </p>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="mt-4"
            onClick={this.retry}
          >
            <RotateCcw className="h-3.5 w-3.5" />
            {this.props.copy.retry}
          </Button>
        </div>
      </div>
    );
  }
}

export function EventPreviewBoundary({
  children,
  resetKey,
}: {
  children: ReactNode;
  resetKey: string | number;
}) {
  const t = useTranslations("eventInGamePreview");
  const copy = useMemo<BoundaryCopy>(
    () => ({
      title: t("previewUnavailableTitle"),
      body: t("previewUnavailableBody"),
      retry: t("previewUnavailableRetry"),
    }),
    [t],
  );
  return (
    <EventPreviewBoundaryInner copy={copy} resetKey={resetKey}>
      {children}
    </EventPreviewBoundaryInner>
  );
}
