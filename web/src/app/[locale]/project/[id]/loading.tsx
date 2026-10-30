export default function ProjectSectionLoading() {
  return (
    <div className="h-full min-h-0 overflow-hidden p-6">
      <div className="flex h-full gap-6">
        <div className="w-72 shrink-0 space-y-1.5">
          <div className="h-8 w-40 rounded-md bg-muted/60 animate-pulse mb-4" />
          {Array.from({ length: 8 }).map((_, i) => (
            <div
              key={`skel-${i}`}
              className="flex items-center justify-between px-3 py-2 rounded-md"
            >
              <div className="flex items-center gap-2 flex-1 min-w-0">
                <div className="h-3 w-8 rounded bg-muted/50 animate-pulse" />
                <div className="h-3 w-32 rounded bg-muted/60 animate-pulse" />
              </div>
              <div className="h-3 w-12 rounded bg-muted/40 animate-pulse" />
            </div>
          ))}
        </div>
        <div className="flex-1 min-w-0 rounded-lg border border-border bg-card/40 animate-pulse" />
      </div>
    </div>
  );
}
