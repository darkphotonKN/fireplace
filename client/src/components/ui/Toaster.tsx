'use client';

import { useEffect, useState } from 'react';
import { useToast, type ToastProps } from './use-toast';

function ToastItem({ toast, onDismiss }: { toast: ToastProps; onDismiss: (id: string) => void }) {
  const [isVisible, setIsVisible] = useState(false);
  const [isLeaving, setIsLeaving] = useState(false);

  useEffect(() => {
    // Trigger enter animation
    requestAnimationFrame(() => setIsVisible(true));

    // Auto-dismiss after 5 seconds
    const timer = setTimeout(() => {
      setIsLeaving(true);
      setTimeout(() => onDismiss(toast.id), 300);
    }, 5000);

    return () => clearTimeout(timer);
  }, [toast.id, onDismiss]);

  const handleClose = () => {
    setIsLeaving(true);
    setTimeout(() => onDismiss(toast.id), 300);
  };

  const isLeft = toast.position === 'bottom-left';

  return (
    <div
      className={`
        flex items-start gap-3 w-80 p-4 rounded-lg shadow-lg
        border-l-4 border-[rgb(247,111,83)]
        bg-[#ebe8d8] dark:bg-[#2a2a2a]
        text-[#2e2e2e] dark:text-[#d1cfc0]
        transition-all duration-300 ease-out
        ${isVisible && !isLeaving
          ? 'opacity-100 translate-y-0'
          : isLeft
            ? 'opacity-0 -translate-x-4'
            : 'opacity-0 translate-x-4'
        }
      `}
    >
      <div className="flex-1 min-w-0">
        {toast.title && (
          <p className="text-sm font-medium">{toast.title}</p>
        )}
        {toast.description && (
          <p className="text-xs mt-1 opacity-70">{toast.description}</p>
        )}
      </div>
      <button
        onClick={handleClose}
        className="shrink-0 p-0.5 rounded hover:bg-foreground/10 transition-colors text-foreground/40 hover:text-foreground/70"
      >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className="w-3.5 h-3.5">
          <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
        </svg>
      </button>
    </div>
  );
}

export default function Toaster() {
  const { toasts, dismiss } = useToast();

  const bottomRight = toasts.filter((t) => t.position !== 'bottom-left');
  const bottomLeft = toasts.filter((t) => t.position === 'bottom-left');

  return (
    <>
      {/* Bottom-right zone */}
      {bottomRight.length > 0 && (
        <div className="fixed bottom-6 right-6 z-50 flex flex-col-reverse gap-3">
          {bottomRight.map((t) => (
            <ToastItem key={t.id} toast={t} onDismiss={dismiss} />
          ))}
        </div>
      )}

      {/* Bottom-left zone */}
      {bottomLeft.length > 0 && (
        <div className="fixed bottom-6 left-6 z-50 flex flex-col-reverse gap-3">
          {bottomLeft.map((t) => (
            <ToastItem key={t.id} toast={t} onDismiss={dismiss} />
          ))}
        </div>
      )}
    </>
  );
}
