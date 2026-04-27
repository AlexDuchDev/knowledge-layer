import { redirect } from "next/navigation";

export default function ControlPlaneNewFeedRedirect() {
  redirect("/source-feeds?from=cp");
}
