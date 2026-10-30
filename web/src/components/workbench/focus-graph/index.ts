import { FocusGraphNode } from "./FocusGraphNode";
import { FocusPrereqEdge } from "./FocusPrereqEdge";
import { FocusMutexEdge } from "./FocusMutexEdge";

export { FocusGraphNode, FocusPrereqEdge, FocusMutexEdge };

export const focusNodeTypes = {
  focusNode: FocusGraphNode,
} as const;

export const focusEdgeTypes = {
  prereqEdge: FocusPrereqEdge,
  mutexEdge: FocusMutexEdge,
} as const;
