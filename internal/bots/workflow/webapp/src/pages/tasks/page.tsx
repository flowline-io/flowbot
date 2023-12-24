import { columns } from "./components/columns"
import { DataTable } from "./components/data-table"
import {Button} from "@/components/ui/button.tsx";
import {Link} from "react-router-dom";
import {useQuery} from "@tanstack/react-query";
import {Client} from "@/util/client.ts";

export default function TaskPage() {

  // Queries
  const query = useQuery({
    queryKey: ['workflows'], queryFn: () => {
      return Client().workflow.getWorkflowWorkflows()
    }
  })

  return (
    <>
      <div className="hidden h-full flex-1 flex-col space-y-8 p-8 md:flex">
        <div className="flex items-center justify-between space-y-2">
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Workflow</h2>
            <p className="text-muted-foreground">
              Here&apos;s a list of your workflows
            </p>
          </div>
          <div className="flex items-center space-x-2">
            <Link to="/script"><Button>+</Button></Link>
          </div>
        </div>
        {query.data?.data ? <DataTable data={query.data?.data} columns={columns} /> : <div className="text-center">Empty</div>}
      </div>
    </>
  )
}
