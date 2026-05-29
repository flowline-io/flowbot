// Alpine.js shared data store
document.addEventListener("alpine:init", () => {
  Alpine.store("app", {
    open: false,
  });
});
