export function ErrorState({ title, message }: { title?: string; message: string }) {
  return (
    <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900" role="alert">
      {title ? <p className="font-medium">{title}</p> : null}
      <p className={title ? "mt-1" : ""}>{message}</p>
    </div>
  );
}
