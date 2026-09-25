import "@testing-library/jest-dom/vitest";

// jsdom does not implement matchMedia; next-themes (used by the app's
// Chakra Provider) needs it to determine light/dark preference.
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

// jsdom does not implement ResizeObserver; Chakra's popper-positioned
// components (Tooltip, Menu, Select, ...) use it to track anchor size.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
window.ResizeObserver = ResizeObserverStub;
