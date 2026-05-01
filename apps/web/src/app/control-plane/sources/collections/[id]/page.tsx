import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { CollectionDetailClient } from "@/components/manual-upload/CollectionDetailClient";

export default async function ManualCollectionDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Collection"
      description="Add content and review the artifacts indexed under this collection."
      screenType="Detail"
      crossLinks={[
        { href: "/control-plane/sources/collections", label: "All collections" },
        { href: "/control-plane/sources", label: "Sources hub" },
      ]}
    >
      <CollectionDetailClient feedId={id} />
    </CpScaffold>
  );
}
