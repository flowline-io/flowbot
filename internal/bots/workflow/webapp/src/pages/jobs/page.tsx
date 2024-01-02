import { columns } from "./components/columns"
import { DataTable } from "./components/data-table"
import {Link, useParams} from "react-router-dom";
import {useQuery} from "@tanstack/react-query";
import {Client} from "@/util/client.ts";

export default function JobPage() {

  let {id} = useParams();

  // Queries
  const query = useQuery({
    queryKey: ['jobs'], queryFn: () => {
      return Client().workflow.getWorkflowWorkflowJobs(parseInt(id))
    }
  })
  const workflow = useQuery({
    queryKey: ['workflow'], queryFn: () => {
      return Client().workflow.getWorkflowWorkflow(parseInt(id))
    }
  })

  return (
    <>
      <div className="hidden h-full flex-1 flex-col space-y-8 p-8 md:flex">
        <div className="flex items-center justify-between space-y-2">
          <div>
            <h2 className="text-2xl font-bold tracking-tight">{workflow.data?.data.name}'s jobs</h2>
            <p className="text-muted-foreground">
              Here&apos;s a list of jobs
            </p>
          </div>
          <div className="flex items-center space-x-2">
            <Link to="/">back to workflows</Link>
          </div>
        </div>
        {query.data?.data ? <DataTable data={query.data?.data} columns={columns} /> : <div className="text-center">Empty</div>}
      </div>
    </>
  )
}
