import { ControlPlaneShell } from "@/components/ControlPlaneShell";

export default function ControlPlaneLayout({ children }: { children: React.ReactNode }) {
  return <ControlPlaneShell>{children}</ControlPlaneShell>;
}
