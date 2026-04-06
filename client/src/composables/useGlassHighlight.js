import { onMounted, onUnmounted, ref } from 'vue';

/**
 * Tracks the mouse position relative to a target element and updates
 * CSS custom properties --highlight-x and --highlight-y so the
 * radial-gradient specular highlight follows the cursor.
 *
 * Usage:
 *   const { elementRef } = useGlassHighlight();
 *   <div ref="elementRef" class="glass"> ... </div>
 */
export function useGlassHighlight() {
  const elementRef = ref(null);
  let rafId = null;

  function onMouseMove(e) {
    if (!elementRef.value) return;

    // Cancel any pending frame to avoid piling up
    if (rafId) cancelAnimationFrame(rafId);

    rafId = requestAnimationFrame(() => {
      const rect = elementRef.value.getBoundingClientRect();
      const x = ((e.clientX - rect.left) / rect.width) * 100;
      const y = ((e.clientY - rect.top) / rect.height) * 100;
      elementRef.value.style.setProperty('--highlight-x', `${x}%`);
      elementRef.value.style.setProperty('--highlight-y', `${y}%`);
    });
  }

  function onMouseLeave() {
    if (!elementRef.value) return;
    // Reset to default resting position
    elementRef.value.style.setProperty('--highlight-x', '30%');
    elementRef.value.style.setProperty('--highlight-y', '20%');
  }

  onMounted(() => {
    if (elementRef.value) {
      elementRef.value.addEventListener('mousemove', onMouseMove);
      elementRef.value.addEventListener('mouseleave', onMouseLeave);
    }
  });

  onUnmounted(() => {
    if (rafId) cancelAnimationFrame(rafId);
    if (elementRef.value) {
      elementRef.value.removeEventListener('mousemove', onMouseMove);
      elementRef.value.removeEventListener('mouseleave', onMouseLeave);
    }
  });

  return { elementRef };
}
