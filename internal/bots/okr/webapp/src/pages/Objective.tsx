import {Col, Input, List, Progress, Row} from "antd";
import React from "react";
import {PlusCircleOutlined} from "@ant-design/icons";
import {useQuery} from "@tanstack/react-query";
import {Client} from "../util/client";
import {useParams} from "react-router-dom";
import {model_KeyResult} from "../client";

const list = [
  {
    title: 'Ant Design Title 1',
    id: "a"
  },
  {
    title: 'Ant Design Title 2',
    id: "b"
  },
  {
    title: 'Ant Design Title 3',
    id: "d"
  },
  {
    title: 'Ant Design Title 4',
    id: "d"
  },
];

function Objective() {

  let {sequence} = useParams();

  // Queries
  const {data} = useQuery({
    queryKey: ['objective'],
    queryFn: () => {
      return Client().okr.getOkrObjective(Number(sequence))
    }
  })

  return (
    <div className="app objective">
      <h1>{data?.data?.title}</h1>

      <h2>进度</h2>
      <div className="item">
        <Progress percent={30}/>
      </div>

      <h2>关键结果</h2>
      <div className="item">
        <Row>
          {data?.data?.key_results?.map((item: model_KeyResult) => (
            <Col span={12}>
              <div className="kr">
                <h3>{item.title}</h3>
                <div className="progress">
                  <div>{item.current_value} -‣ {item.target_value}</div>
                  <PlusCircleOutlined/>
                </div>
              </div>
            </Col>
          ))}
        </Row>
      </div>

      <h2>动机</h2>
      <div className="item">
        <List
          dataSource={list}
          renderItem={(item, index) => (
            <List.Item>
              <List.Item.Meta
                title={<>{index + 1}. {item.title}</>}
              />
            </List.Item>
          )}
        />
      </div>

      <h2>可行性</h2>
      <div className="item">
        <List
          dataSource={list}
          renderItem={(item, index) => (
            <List.Item>
              <List.Item.Meta
                title={<>{index + 1}. {item.title}</>}
              />
            </List.Item>
          )}
        />
      </div>

      <h2>备忘</h2>
      <div>
        <Input.TextArea value={data?.data?.memo} rows={10}/>
      </div>
    </div>
  )
}

export default Objective;