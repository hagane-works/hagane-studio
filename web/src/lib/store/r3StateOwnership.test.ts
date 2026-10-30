import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  charactersApi,
  decisionsApi,
  eventsApi,
  ideasApi,
  statesApi,
  technologiesApi,
} from "@/lib/api";
import { useCharacterStore } from "./characterStore";
import { useDecisionStore } from "./decisionStore";
import { useEventStore } from "./eventStore";
import { useIdeaStore } from "./ideaStore";
import { useStateStore } from "./stateStore";
import { useTechnologyStore } from "./technologyStore";

vi.mock("@/lib/api", () => ({
  charactersApi: { list: vi.fn(), preview: vi.fn() },
  decisionsApi: { listCategories: vi.fn(), list: vi.fn(), preview: vi.fn() },
  eventsApi: { listNamespaces: vi.fn(), list: vi.fn(), preview: vi.fn() },
  ideasApi: { list: vi.fn(), preview: vi.fn() },
  statesApi: { list: vi.fn() },
  technologiesApi: { list: vi.fn(), preview: vi.fn() },
}));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

describe("R3 list and preview ownership", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useCharacterStore.setState({
      characters: [],
      selectedId: null,
      scriptPreview: "",
      loading: false,
      error: null,
    });
    useDecisionStore.setState({
      categories: [],
      decisions: [],
      selectedCatId: null,
      selectedDecisionId: null,
      scriptPreview: "",
      loading: false,
      error: null,
    });
    useEventStore.setState({
      namespaces: [],
      events: [],
      selectedNsId: null,
      selectedEventId: null,
      scriptPreview: "",
      loading: false,
      error: null,
    });
    useIdeaStore.setState({
      ideas: [],
      selectedId: null,
      scriptPreview: "",
      loading: false,
      error: null,
      filterType: "",
    });
    useStateStore.setState({
      states: [],
      selectedId: null,
      loading: false,
      error: null,
      lastProvinceMove: null,
      provinceMoveFailure: null,
    });
    useTechnologyStore.setState({
      technologies: [],
      selectedId: null,
      scriptPreview: "",
      loading: false,
      error: null,
      selectedTechLine: "infantry",
      historyUndo: [],
      historyRedo: [],
    });
  });

  it("keeps project B's character list when deferred project A finishes late", async () => {
    const a = deferred<unknown[]>();
    const b = deferred<unknown[]>();
    vi.mocked(charactersApi.list)
      .mockReturnValueOnce(a.promise as ReturnType<typeof charactersApi.list>)
      .mockReturnValueOnce(b.promise as ReturnType<typeof charactersApi.list>);

    const oldRequest = useCharacterStore.getState().fetchCharacters(701);
    const newRequest = useCharacterStore.getState().fetchCharacters(702);
    const projectB = [{ id: 2, name: "B" }];
    b.resolve(projectB);
    await newRequest;
    a.resolve([{ id: 1, name: "A" }]);
    await oldRequest;

    expect(useCharacterStore.getState().characters).toEqual(projectB);
    expect(useCharacterStore.getState().selectedId).toBeNull();
  });

  it("does not let a deferred decision preview for A replace B", async () => {
    vi.mocked(decisionsApi.listCategories).mockResolvedValue([] as never);
    await useDecisionStore.getState().fetchCategories(703);
    const a = deferred<{ script: string }>();
    const b = deferred<{ script: string }>();
    vi.mocked(decisionsApi.preview)
      .mockReturnValueOnce(a.promise as ReturnType<typeof decisionsApi.preview>)
      .mockReturnValueOnce(
        b.promise as ReturnType<typeof decisionsApi.preview>,
      );

    useDecisionStore.getState().selectDecision(11);
    const oldPreview = useDecisionStore.getState().fetchPreview(703, 11);
    useDecisionStore.getState().selectDecision(12);
    const newPreview = useDecisionStore.getState().fetchPreview(703, 12);
    b.resolve({ script: "B preview" });
    await newPreview;
    a.resolve({ script: "A preview" });
    await oldPreview;

    expect(useDecisionStore.getState().selectedDecisionId).toBe(12);
    expect(useDecisionStore.getState().scriptPreview).toBe("B preview");
  });

  it("drops stale event and idea lists after their selection bucket changes", async () => {
    vi.mocked(eventsApi.listNamespaces).mockResolvedValue([] as never);
    await useEventStore.getState().fetchNamespaces(704);
    useEventStore.getState().selectNamespace(1);
    const oldEvents = deferred<unknown[]>();
    const newEvents = deferred<unknown[]>();
    vi.mocked(eventsApi.list)
      .mockReturnValueOnce(
        oldEvents.promise as ReturnType<typeof eventsApi.list>,
      )
      .mockReturnValueOnce(
        newEvents.promise as ReturnType<typeof eventsApi.list>,
      );
    const a = useEventStore.getState().fetchEvents(704, 1);
    useEventStore.getState().selectNamespace(2);
    const b = useEventStore.getState().fetchEvents(704, 2);
    newEvents.resolve([{ id: 22, namespace_id: 2 }]);
    await b;
    oldEvents.resolve([{ id: 11, namespace_id: 1 }]);
    await a;
    expect(useEventStore.getState().events).toEqual([
      { id: 22, namespace_id: 2 },
    ]);

    const oldIdeas = deferred<unknown[]>();
    const newIdeas = deferred<unknown[]>();
    vi.mocked(ideasApi.list)
      .mockReturnValueOnce(oldIdeas.promise as ReturnType<typeof ideasApi.list>)
      .mockReturnValueOnce(
        newIdeas.promise as ReturnType<typeof ideasApi.list>,
      );
    useIdeaStore.getState().setFilterType("country" as never);
    const oldIdeaRequest = useIdeaStore.getState().fetchIdeas(704);
    useIdeaStore.getState().setFilterType("minister" as never);
    const newIdeaRequest = useIdeaStore.getState().fetchIdeas(704);
    newIdeas.resolve([{ id: 32, idea_type: "minister" }]);
    await newIdeaRequest;
    oldIdeas.resolve([{ id: 31, idea_type: "country" }]);
    await oldIdeaRequest;
    expect(useIdeaStore.getState().ideas).toEqual([
      { id: 32, idea_type: "minister" },
    ]);
  });

  it("keeps project B's states and selected technology preview when A completes late", async () => {
    const oldStates = deferred<unknown[]>();
    const newStates = deferred<unknown[]>();
    vi.mocked(statesApi.list)
      .mockReturnValueOnce(
        oldStates.promise as ReturnType<typeof statesApi.list>,
      )
      .mockReturnValueOnce(
        newStates.promise as ReturnType<typeof statesApi.list>,
      );
    const a = useStateStore.getState().fetchStates(705);
    const b = useStateStore.getState().fetchStates(706);
    newStates.resolve([{ id: 6, state_id: 6 }]);
    await b;
    oldStates.resolve([{ id: 5, state_id: 5 }]);
    await a;
    expect(useStateStore.getState().states).toEqual([{ id: 6, state_id: 6 }]);

    vi.mocked(technologiesApi.list).mockResolvedValue([] as never);
    await useTechnologyStore.getState().fetchTechnologies(706, "infantry");
    const oldPreview = deferred<{ script: string }>();
    const newPreview = deferred<{ script: string }>();
    vi.mocked(technologiesApi.preview)
      .mockReturnValueOnce(
        oldPreview.promise as ReturnType<typeof technologiesApi.preview>,
      )
      .mockReturnValueOnce(
        newPreview.promise as ReturnType<typeof technologiesApi.preview>,
      );
    useTechnologyStore.getState().selectTech(41);
    const techA = useTechnologyStore
      .getState()
      .fetchPreview(706, "infantry", 41);
    useTechnologyStore.getState().selectTech(42);
    const techB = useTechnologyStore
      .getState()
      .fetchPreview(706, "infantry", 42);
    newPreview.resolve({ script: "technology B" });
    await techB;
    oldPreview.resolve({ script: "technology A" });
    await techA;
    expect(useTechnologyStore.getState().scriptPreview).toBe("technology B");
  });
});
