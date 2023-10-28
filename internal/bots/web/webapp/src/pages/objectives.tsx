import FlexContainer from "@/components/flex-container";
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query";
import {Client} from "@/util/client";
import {model_Objective} from "@/client";
import {Link} from "react-router-dom";
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card";
import {PersonIcon} from "@radix-ui/react-icons";
import {Progress} from "@/components/ui/progress";


export default function ObjectivesPage() {

  // Access the client
  const queryClient = useQueryClient()

  // Queries
  const query = useQuery({
    queryKey: ['objectives'], queryFn: () => {
      return Client().okr.getOkrObjectives()
    }
  })

  // Mutations
  const mutation = useMutation({
    mutationFn: Client().okr.postOkrObjective,
    onSuccess: () => {
      // Invalidate and refetch
      queryClient.invalidateQueries({queryKey: ['objectives']})
    },
  })

  console.log(query.data?.data)

  return (
    <>
      <div className="items-start justify-center gap-6 rounded-lg p-8 md:grid lg:grid-cols-2 xl:grid-cols-3">
        <div className="col-span-2 grid items-start gap-6 lg:col-span-1">
          <FlexContainer>
            <Card>
              <CardHeader className="pb-3">
                <CardTitle>All Objectives</CardTitle>
                <CardDescription>
                  In progress
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-1">
                {query.data?.data.map((item: model_Objective) => (
                  <Link to={`obj/${item.sequence}`}>
                    <div key={`${item.id}`}
                         className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                      <PersonIcon className="mt-px h-5 w-5"/>
                      <div className="space-y-1">
                        <div className="text-sm font-medium leading-none">{item.title}</div>
                        <div className="text-sm text-muted-foreground">
                          2023/7/9 ~ 2023/8/9
                        </div>
                        <div className="text-sm">
                          <Progress value={30} className="w-[60%]"/>
                        </div>
                      </div>
                    </div>
                  </Link>
                ))}
              </CardContent>
              <Link to="obj">Create objective</Link>
            </Card>
          </FlexContainer>
        </div>
      </div>
    </>
  )
}
