import FlexContainer from "@/components/flex-container";
import {useQuery} from "@tanstack/react-query";
import {Client} from "@/util/client";
import {model_Objective} from "@/client";
import {Link} from "react-router-dom";
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card";
import {CubeIcon} from "@radix-ui/react-icons";
import {Progress} from "@/components/ui/progress";
import {Button} from "@/components/ui/button.tsx";

export default function ObjectivesPage() {

  // Queries
  const query = useQuery({
    queryKey: ['objectives'], queryFn: () => {
      return Client().okr.getOkrObjectives()
    }
  })

  return (
    <>
      <div className="items-start justify-center gap-6 rounded-lg p-8 grid grid-cols-1">
        <div className="grid col-span-1 items-start gap-6">
          <FlexContainer>
            <Card>
              <CardHeader>
                <CardTitle>All Objectives</CardTitle>
                <CardDescription>
                  In progress
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-6">
                {query.data?.data.map((item: model_Objective) => (
                  <Link key={`${item.id}`} to={`obj/${item.sequence}`}>
                    <div className="w-[600px] -mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                      <CubeIcon className="mt-px h-5 w-5"/>
                      <div className="w-[600px]">
                        <div className="text-sm font-medium leading-none mb-2">{item.title}</div>
                        <div className="text-sm text-muted-foreground mb-2">
                          {item.is_plan ? `${item.plan_start?.substring(0, 10)} ~ ${item.plan_end?.substring(0, 10)}` : "-"}
                        </div>
                        <div className="text-sm">
                          <Progress value={item.progress} className="w-[100%]"/>
                        </div>
                      </div>
                    </div>
                  </Link>
                ))}
              </CardContent>
              <Link to="obj">
                <Button variant="secondary" size="sm" className="m-3 float-right clear-both">Create objective</Button>
              </Link>
            </Card>
          </FlexContainer>
        </div>
      </div>
    </>
  )
}
