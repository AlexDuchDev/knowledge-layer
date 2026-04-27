export function LoadingState({ message = "Loading…" }: { message?: string }) {
  return (
    <div className="flex items-center justify-center py-16 text-sm text-neutral-500" role="status">
      {message}
    </div>
  );
}
