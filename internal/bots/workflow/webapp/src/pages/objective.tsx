import FlexContainer from "@/components/flex-container";
import {Card, CardContent, CardHeader, CardTitle} from "@/components/ui/card";
import {CubeIcon} from "@radix-ui/react-icons";
import {Progress} from "@/components/ui/progress";
import {Textarea} from "@/components/ui/textarea";
import {Client} from "@/util/client";
import {useQuery} from "@tanstack/react-query";
import {Link, useParams} from "react-router-dom";
import { model_KeyResult } from "@/client";
import {Button} from "@/components/ui/button.tsx";

export default function ObjectivePage() {

  let {sequence} = useParams();

  // Queries
  const {data } = useQuery({
    queryKey: [`objective-${sequence}`],
    queryFn: () => {
      return Client().okr.getOkrObjective(Number(sequence))
    }
  })

  return (
    <>
      <div className="items-start justify-center gap-6 rounded-lg p-8 grid grid-cols-1">
        <div className="grid col-span-1 items-start gap-6">
          <FlexContainer>
            <Card>
              <CardHeader>
                <CardTitle>{ data?.data.title }</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-6">
                <div className="w-[600px]">
                  <div className="mb-3">进度</div>
                  <div className="grid gap-1 mb-3">
                    <Progress value={ data?.data.progress } className="w-[100%]"/>
                  </div>

                  <div className="mb-3">动机</div>
                  <div className="grid gap-1 mb-3">
                    <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                      <CubeIcon className="mt-px h-5 w-5"/>
                      <div className="space-y-1">
                        <div className="text-sm font-medium leading-none">title</div>
                      </div>
                    </div>
                  </div>

                  <div className="mb-3">可行性</div>
                  <div className="grid gap-1 mb-3">
                    <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                      <CubeIcon className="mt-px h-5 w-5"/>
                      <div className="space-y-1">
                        <div className="text-sm font-medium leading-none">title</div>
                      </div>
                    </div>
                  </div>

                  <div className="mb-3">关键结果</div>
                  <div className="grid gap-1 mb-3">
                    {data?.data.key_results.map((item: model_KeyResult) => (
                      <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                        <CubeIcon className="mt-px h-5 w-5"/>
                        <div className="space-y-1">
                          <div className="text-sm font-medium leading-none">{item.title}</div>
                          <div className="text-sm text-muted-foreground">
                            {item.initial_value} - {item.target_value}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>

                  <div className="mb-3">备忘</div>
                  <div className="grid gap-1 mb-3">
                    <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                      <Textarea readOnly={true}>{data?.data.memo}</Textarea>
                    </div>
                  </div>
                </div>
                <Link to="/">
                  <Button variant="secondary" size="sm" className="m-1 float-right clear-both">返回</Button>
                </Link>
              </CardContent>
            </Card>
          </FlexContainer>
        </div>
      </div>
    </>
  )
}
