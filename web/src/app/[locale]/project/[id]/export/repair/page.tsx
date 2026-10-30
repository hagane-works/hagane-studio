"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { useParams } from "@/platform/navigation";
import { toast } from "sonner";
import { LoaderCircle } from "lucide-react";

import { exportApi } from "@/lib/api";
import { useProjectStore } from "@/lib/store/projectStore";
import {
  buildRepairQueue,
  findRepairIndexByHash,
  nextUnresolvedIndex,
  reconcileRepairQueue,
  repairHash,
  type RepairQueueItem,
} from "@/lib/exportRepairQueue";
import { RepairOriginBanner } from "@/components/export/repair/RepairOriginBanner";
import { RepairQueueList } from "@/components/export/repair/RepairQueueList";
import { RepairFindingPanel } from "@/components/export/repair/RepairFindingPanel";
import { RepairOutcomeCard } from "@/components/export/repair/RepairOutcomeCard";

export default function ExportRepairPage() {
  const t = useTranslations("export");
  const params = useParams();
  const projectId = Number(params.id);
  const currentProject = useProjectStore((s) => s.currentProject);

  const projectReady = currentProject?.id === projectId;

  const includeImported = currentProject?.is_loaded_existing_mod ?? false;

  const [items, setItems] = useState<RepairQueueItem[] | null>(null);
  const [loadFailed, setLoadFailed] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const [rechecking, setRechecking] = useState(false);
  const [recheckFailed, setRecheckFailed] = useState(false);
  const hashApplied = useRef(false);
  const panelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!projectId || !projectReady) return;
    let alive = true;
    hashApplied.current = false;
    setItems(null);
    setLoadFailed(false);
    exportApi
      .check(projectId, includeImported, [])
      .then((result) => {
        if (!alive) return;
        setItems(buildRepairQueue(result.issues ?? []));
      })
      .catch(() => {
        if (alive) setLoadFailed(true);
      });
    return () => {
      alive = false;
    };
  }, [projectId, projectReady, includeImported]);

  const focusIndex = useCallback((index: number, list: RepairQueueItem[]) => {
    setActiveIndex(index);
    const item = list[index];
    if (item && typeof window !== "undefined") {
      const { pathname, search } = window.location;
      window.history.replaceState(
        null,
        "",
        `${pathname}${search}#${repairHash(item.finding.issue)}`,
      );
    }
  }, []);

  useEffect(() => {
    if (!items || hashApplied.current) return;
    hashApplied.current = true;
    const fromHash =
      typeof window === "undefined"
        ? -1
        : findRepairIndexByHash(items, window.location.hash);
    if (fromHash >= 0) {
      setActiveIndex(fromHash);
      return;
    }
    const open = nextUnresolvedIndex(items, 0);
    setActiveIndex(open >= 0 ? open : 0);
  }, [items]);

  useEffect(() => {
    if (!items) return;
    const onHashChange = () => {
      const index = findRepairIndexByHash(items, window.location.hash);
      if (index >= 0) setActiveIndex(index);
    };
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, [items]);

  const handleSelect = useCallback(
    (index: number) => {
      if (!items) return;
      focusIndex(index, items);
      panelRef.current?.scrollIntoView({
        behavior: "smooth",
        block: "nearest",
      });
    },
    [items, focusIndex],
  );

  const handleRecheck = useCallback(async () => {
    if (!items || rechecking) return;
    setRechecking(true);
    setRecheckFailed(false);
    try {
      const result = await exportApi.check(projectId, includeImported, []);
      const reconciled = reconcileRepairQueue(items, result.issues ?? []);
      setItems(reconciled.items);
      if (reconciled.newlyResolved.length > 0) {
        toast.success(
          t("ui.repair.toastResolved", {
            count: reconciled.newlyResolved.length,
          }),
        );
      } else if (reconciled.added.length === 0) {
        toast(t("ui.repair.toastUnchanged"));
      }
      if (reconciled.added.length > 0) {
        toast(t("ui.repair.toastNew", { count: reconciled.added.length }));
      }
      const open = nextUnresolvedIndex(reconciled.items, activeIndex);
      if (open >= 0) focusIndex(open, reconciled.items);
    } catch {
      setRecheckFailed(true);
    } finally {
      setRechecking(false);
    }
  }, [
    items,
    rechecking,
    projectId,
    includeImported,
    activeIndex,
    focusIndex,
    t,
  ]);

  const safeIndex = useMemo(() => {
    if (!items || items.length === 0) return 0;
    return Math.min(Math.max(activeIndex, 0), items.length - 1);
  }, [items, activeIndex]);
  const activeItem = items && items.length > 0 ? items[safeIndex] : null;
  const nextOpen = items ? nextUnresolvedIndex(items, safeIndex + 1) : -1;
  const hasNext = nextOpen >= 0 && nextOpen !== safeIndex;
  const allResolved =
    items !== null && items.length > 0 && items.every((item) => item.resolved);

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto max-w-5xl space-y-4 p-6">
        <div>
          <h2 className="mb-1 text-lg font-semibold">
            {t("ui.repair.pageTitle")}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t("ui.repair.pageIntro")}
          </p>
        </div>

        <RepairOriginBanner projectId={projectId} />

        {loadFailed ? (
          <RepairOutcomeCard outcome="failed" projectId={projectId} />
        ) : items === null ? (
          <div className="flex items-center gap-2.5 rounded-lg border border-border bg-card px-4 py-6 text-sm text-muted-foreground">
            <LoaderCircle className="h-4 w-4 animate-spin" aria-hidden />
            {t("ui.repair.loading")}
          </div>
        ) : items.length === 0 ? (
          <RepairOutcomeCard outcome="empty" projectId={projectId} />
        ) : allResolved ? (
          <RepairOutcomeCard outcome="done" projectId={projectId} />
        ) : (
          <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(240px,290px)_minmax(0,1fr)]">
            <RepairQueueList
              projectId={projectId}
              items={items}
              activeIndex={safeIndex}
              onSelect={handleSelect}
            />
            <div ref={panelRef}>
              {activeItem ? (
                <RepairFindingPanel
                  projectId={projectId}
                  item={activeItem}
                  index={safeIndex}
                  total={items.length}
                  rechecking={rechecking}
                  recheckFailed={recheckFailed}
                  onRecheck={() => void handleRecheck()}
                  onNext={() => handleSelect(nextOpen)}
                  hasNext={hasNext}
                />
              ) : null}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
