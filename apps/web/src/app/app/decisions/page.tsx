import { redirect } from "next/navigation";

export default function LegacyAppDecisionsRedirect() {
  redirect("/decisions");
}
