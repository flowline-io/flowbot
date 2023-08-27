import {Col, Input, List, Progress, Row} from "antd";
import React from "react";
import {PlusCircleOutlined} from "@ant-design/icons";

const data = [
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
  return (
    <div className="app objective">
      <h1>Objective</h1>

      <h2>进度</h2>
      <div className="item">
        <Progress percent={30}/>
      </div>

      <h2>关键结果</h2>
      <div className="item">
        <Row>
          <Col span={12}>
            <div className="kr">
              <h3>功能一</h3>
              <div className="progress">
                <div>7 -‣ 10</div>
                <PlusCircleOutlined/>
              </div>
            </div>
          </Col>
          <Col span={12}>
            <div className="kr">
              <h3>功能一</h3>
              <div className="progress">
                <div>7 -‣ 10</div>
                <PlusCircleOutlined/>
              </div>
            </div>
          </Col>
        </Row>
      </div>

      <h2>动机</h2>
      <div className="item">
        <List
          dataSource={data}
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
          dataSource={data}
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
        <Input.TextArea value="在原有的系统升级并完善" rows={10}/>
      </div>
    </div>
  )
}

export default Objective;