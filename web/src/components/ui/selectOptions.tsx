import * as React from "react";

export type SelectOpt = {
  value: string;
  node: React.ReactNode;
  disabled?: boolean;
};

export type SelectSection = {
  label: React.ReactNode | null;
  options: SelectOpt[];
};

function readOption(child: React.ReactElement): SelectOpt {
  const p = child.props as {
    value?: string | number;
    children?: React.ReactNode;
    disabled?: boolean;
  };
  return {
    value: String(p.value ?? ""),
    node: p.children,
    disabled: p.disabled,
  };
}

export function collectSections(children: React.ReactNode): SelectSection[] {
  const sections: SelectSection[] = [];

  let loose: SelectSection | null = null;

  const pushLoose = (opt: SelectOpt) => {
    if (!loose) {
      loose = { label: null, options: [] };
      sections.push(loose);
    }
    loose.options.push(opt);
  };

  const walk = (node: React.ReactNode) => {
    React.Children.forEach(node, (child) => {
      if (!React.isValidElement(child)) return;
      if (child.type === React.Fragment) {
        walk((child.props as { children?: React.ReactNode }).children);
        return;
      }
      if (child.type === "option") {
        pushLoose(readOption(child));
        return;
      }
      if (child.type === "optgroup") {
        const p = child.props as {
          label?: React.ReactNode;
          children?: React.ReactNode;
        };
        const group: SelectSection = { label: p.label ?? null, options: [] };

        const collectInner = (inner: React.ReactNode) => {
          React.Children.forEach(inner, (c) => {
            if (!React.isValidElement(c)) return;
            if (c.type === React.Fragment) {
              collectInner(
                (c.props as { children?: React.ReactNode }).children,
              );
              return;
            }
            if (c.type === "option") group.options.push(readOption(c));
          });
        };
        collectInner(p.children);

        if (group.options.length > 0) {
          sections.push(group);
          loose = null;
        }
      }
    });
  };
  walk(children);
  return sections;
}
