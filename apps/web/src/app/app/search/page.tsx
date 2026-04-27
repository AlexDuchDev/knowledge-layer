import { redirect } from "next/navigation";

export default function LegacyAppSearchRedirect() {
  redirect("/search");
}
