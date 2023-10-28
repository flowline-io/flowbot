import FlexContainer from "@/components/flex-container";
import {Card, CardContent, CardHeader, CardTitle} from "@/components/ui/card";
import {PersonIcon} from "@radix-ui/react-icons";
import {Progress} from "@/components/ui/progress";
import {Textarea} from "@/components/ui/textarea";

export default function ObjectivePage() {
  return (
    <>
      <div className="items-start justify-center gap-6 rounded-lg p-8 md:grid lg:grid-cols-2 xl:grid-cols-3">
        <div className="col-span-2 grid items-start gap-6 lg:col-span-1">
          <FlexContainer>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle>Objectives 1</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-1">
                <div>进度</div>
                <div className="grid gap-1">
                  <Progress value={30} className="w-[60%]"/>
                </div>

                <div>动机</div>
                <div className="grid gap-1">
                  <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                    <PersonIcon className="mt-px h-5 w-5"/>
                    <div className="space-y-1">
                      <div className="text-sm font-medium leading-none">title</div>
                    </div>
                  </div>
                </div>

                <div>可行性</div>
                <div className="grid gap-1">
                  <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                    <PersonIcon className="mt-px h-5 w-5"/>
                    <div className="space-y-1">
                      <div className="text-sm font-medium leading-none">title</div>
                    </div>
                  </div>
                </div>

                <div>关键结果</div>
                <div className="grid gap-1">
                  <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                    <PersonIcon className="mt-px h-5 w-5"/>
                    <div className="space-y-1">
                      <div className="text-sm font-medium leading-none">title</div>
                      <div className="text-sm text-muted-foreground">
                        0 - 20
                      </div>
                    </div>
                  </div>
                </div>

                <div>备忘</div>
                <div className="grid gap-1">
                  <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                    <Textarea />
                  </div>
                </div>
              </CardContent>
            </Card>
          </FlexContainer>
        </div>
      </div>
    </>
  )
}
