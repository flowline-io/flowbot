import FlexContainer from "@/components/flex-container";

export default function ExamplePage() {
  return (
    <>
      <div className="items-start justify-center gap-6 rounded-lg p-8 grid grid-cols-1">
        <div className="grid col-span-1 items-start gap-6">
          <FlexContainer>
            <div>page</div>
          </FlexContainer>
        </div>
      </div>
    </>
  )
}
