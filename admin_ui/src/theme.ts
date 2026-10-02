import { createSystem, defaultConfig } from "@chakra-ui/react";

export const system = createSystem(defaultConfig, {
  globalCss: {
    body: { fontFeatureSettings: "normal" },
  },
  theme: {
    tokens: {
      fonts: {
        heading: {
          value: `"IBM Plex Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif`,
        },
        body: {
          value: `"IBM Plex Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif`,
        },
        mono: {
          value: `"IBM Plex Mono", SFMono-Regular, Menlo, Consolas, monospace`,
        },
      },
    },
  },
});
