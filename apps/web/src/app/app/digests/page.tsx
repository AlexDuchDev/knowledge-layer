import { redirect } from "next/navigation";

export default function LegacyAppDigestsRedirect() {
  redirect("/insights");
}
