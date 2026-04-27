// Legacy /app/* segment: routes redirect to canonical dash URLs (see next.config.ts).
export default function LegacyAppSegmentLayout({ children }: { children: React.ReactNode }) {
  return children;
}
