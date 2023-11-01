import FlexContainer from "@/components/flex-container";
import {Card, CardContent, CardHeader, CardTitle} from "@/components/ui/card";
import {PersonIcon} from "@radix-ui/react-icons";
import {Progress} from "@/components/ui/progress";
import {Textarea} from "@/components/ui/textarea";
import {Client} from "@/util/client";
import {useQuery} from "@tanstack/react-query";
import {useParams} from "react-router-dom";
import { model_KeyResult } from "@/client";

export default function ObjectivePage() {

  let {sequence} = useParams();

  // Queries
  const {data } = useQuery({
    queryKey: ['objective'],
    queryFn: () => {
      return Client().okr.getOkrObjective(Number(sequence))
    }
  })

  return (
    <>
      <div className="items-start justify-center gap-6 rounded-lg p-8 md:grid lg:grid-cols-2 xl:grid-cols-3">
        <div className="col-span-2 grid items-start gap-6 lg:col-span-1">
          <FlexContainer>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle>{ data?.data.title }</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-1">
                <div>进度</div>
                <div className="grid gap-1">
                  <Progress value={ data?.data.progress } className="w-[60%]"/>
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
                  {data?.data.key_results.map((item: model_KeyResult) => (
                    <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                      <PersonIcon className="mt-px h-5 w-5"/>
                      <div className="space-y-1">
                        <div className="text-sm font-medium leading-none">{item.title}</div>
                        <div className="text-sm text-muted-foreground">
                          {item.initial_value} - {item.target_value}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>

                <div>备忘</div>
                <div className="grid gap-1">
                  <div className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                    <Textarea readOnly={true}>{data?.data.memo}</Textarea>
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
