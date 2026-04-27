import { redirect } from "next/navigation";

export default function LegacyAppProjectsRedirect() {
  redirect("/projects");
}
