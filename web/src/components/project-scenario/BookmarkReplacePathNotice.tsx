import { AlertTriangle, CheckCircle2, Info } from "lucide-react";
import { Button } from "@/components/ui/button";

export interface BookmarkReplacePathNoticeProps {
  bookmarkCount: number;
  title: string;
  description: string;
  singleBookmarkWarning: string;
  singleBookmarkAction: string;
  addSecondBookmarkLabel: string;
  secondBookmarkDraftMessage: string;
  navigationReadyMessage: string;
  isCreatingSecond: boolean;
  showNavigationReady: boolean;
  onAddSecond: () => void;
}

export function BookmarkReplacePathNotice({
  bookmarkCount,
  title,
  description,
  singleBookmarkWarning,
  singleBookmarkAction,
  addSecondBookmarkLabel,
  secondBookmarkDraftMessage,
  navigationReadyMessage,
  isCreatingSecond,
  showNavigationReady,
  onAddSecond,
}: BookmarkReplacePathNoticeProps) {
  if (bookmarkCount <= 0) return null;

  return (
    <div className="space-y-2">
      <section
        role="note"
        aria-label={title}
        className="rounded-md border border-border bg-muted/30 p-3"
      >
        <div className="flex items-start gap-2">
          <Info
            aria-hidden
            className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground"
          />
          <div className="min-w-0">
            <h3 className="text-sm font-medium text-foreground">{title}</h3>
            <p className="mt-1 text-xs leading-snug text-muted-foreground">
              {description}
            </p>
          </div>
        </div>
      </section>

      {showNavigationReady ? (
        <div
          role="status"
          className="rounded-md border border-success/40 bg-success/5 p-3"
        >
          <div className="flex items-start gap-2">
            <CheckCircle2
              aria-hidden
              className="mt-0.5 h-4 w-4 shrink-0 text-success"
            />
            <p className="text-sm font-medium text-foreground">
              {navigationReadyMessage}
            </p>
          </div>
        </div>
      ) : bookmarkCount === 1 ? (
        <div
          role="alert"
          className="rounded-md border border-warning/40 bg-warning/5 p-3"
        >
          <div className="flex items-start gap-2">
            <AlertTriangle
              aria-hidden
              className="mt-0.5 h-4 w-4 shrink-0 text-warning"
            />
            <div className="min-w-0">
              <p className="text-sm font-medium text-foreground">
                {isCreatingSecond
                  ? secondBookmarkDraftMessage
                  : singleBookmarkWarning}
              </p>
              {!isCreatingSecond ? (
                <>
                  <p className="mt-1 text-xs font-medium leading-snug text-foreground">
                    {singleBookmarkAction}
                  </p>
                  <Button
                    type="button"
                    size="sm"
                    className="mt-2"
                    onClick={onAddSecond}
                  >
                    {addSecondBookmarkLabel}
                  </Button>
                </>
              ) : null}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
