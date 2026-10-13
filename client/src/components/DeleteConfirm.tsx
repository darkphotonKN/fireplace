'use client';

/**
 * The plan's one delete confirmation: lifted out of the archived list so a
 * milestone's page asks the same question the same way (FS-043WP R18).
 */
export default function DeleteConfirm({
  description,
  onCancel,
  onConfirm,
}: {
  description: string;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-white/5 backdrop-blur-md border border-white/10 rounded-lg p-6 max-w-md w-full mx-4 animate-slideIn">
        <h3 className="text-lg font-medium mb-2">Delete Task</h3>
        <p className="text-gray-300 mb-4">
          Are you sure you want to permanently delete &quot;{description}&quot;? This
          action cannot be undone.
        </p>
        <div className="flex justify-end space-x-3">
          <button
            onClick={onCancel}
            className="px-4 py-2 rounded bg-white/5 hover:bg-white/10 border border-white/10 text-base"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="px-4 py-2 rounded bg-red-500/10 hover:bg-red-500/20 border border-red-500/20 text-red-500 text-base"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  );
}
