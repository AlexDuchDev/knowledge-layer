import { redirect } from "next/navigation";

export default function LegacyAppExplorerRedirect() {
  redirect("/entities");
}
