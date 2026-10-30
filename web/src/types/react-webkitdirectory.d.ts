import "react";

declare module "react" {
  interface InputHTMLAttributes<T> {
    webkitdirectory?: T extends HTMLInputElement ? string : never;
  }
}
