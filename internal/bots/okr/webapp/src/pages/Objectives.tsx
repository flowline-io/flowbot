import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query";
import {Client} from "../util/client";
import {List, Progress} from "antd";
import {model_Objective} from "../client";
import {Link} from "react-router-dom";
import {CalendarOutlined} from "@ant-design/icons";
import React from "react";

function Objectives() {
  // Access the client
  const queryClient = useQueryClient()

  // Queries
  const query = useQuery({ queryKey: ['objectives'], queryFn: ()=> {return Client().okr.getOkrObjectives()} })

  // Mutations
  const mutation = useMutation({
    mutationFn: Client().okr.postOkrObjective,
    onSuccess: () => {
      // Invalidate and refetch
      queryClient.invalidateQueries({ queryKey: ['objectives'] })
    },
  })

  return (
    <div className="app objectives">
      <h1>All Objectives</h1>
      <h2>In progress</h2>
      <List
        className="list"
        dataSource={query.data?.data}
        renderItem={(item : model_Objective, index) => (
          <List.Item>
            <List.Item.Meta
              title={<><Link to={`obj/${item.sequence}`}>{item.title}</Link></>}
              description={
                <>
                  <div><CalendarOutlined/> 2023/7/9 ~ 2023/8/9</div>
                  <Progress percent={30}/>
                </>
              }
            />
          </List.Item>
        )}
        loading={query.isLoading}
      />
      <div><Link to="obj">Create objective</Link></div>
    </div>
  )
}

export default Objectives;