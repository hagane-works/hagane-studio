import type { ReactNode } from "react";
import Link from "@/platform/link";
import styles from "./SiteHeader.module.css";
import { SiteHeaderNavigation } from "./SiteHeaderNavigation";

export { styles as siteHeaderStyles };

export function SiteHeader({
  brandHref,
  brandLabel,
  navLabel,
  nav,
  tools,
  containerClassName,
  children,
}: {
  brandHref: string;
  brandLabel: string;
  navLabel: string;
  nav: ReactNode;
  tools: ReactNode;
  containerClassName: string;

  children?: ReactNode;
}) {
  return (
    <header className={"public-site " + styles.header}>
      <div className={containerClassName + " " + styles.inner}>
        <Link
          href={brandHref}
          aria-label={brandLabel}
          className={styles.brand + " ui-interactive ui-focus-ring"}
        >
          <img
            src="/brand/hagane-icon-96.png"
            alt=""
            width={40}
            height={40}
            decoding="async"
            data-logo-slot="brand"
            className={styles.logo}
          />
          <span>
            <span lang="zh-Hans" className={styles.brandName}>
              钢之工坊
            </span>
            <span lang="en" className={styles.brandEnglish}>
              Hagane Works
            </span>
          </span>
        </Link>
        <SiteHeaderNavigation label={navLabel}>{nav}</SiteHeaderNavigation>
        <div className={styles.tools}>{tools}</div>
      </div>
      {children}
    </header>
  );
}
