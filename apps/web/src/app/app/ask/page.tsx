import { redirect } from "next/navigation";

export default function LegacyAppAskRedirect() {
  redirect("/ask");
}
