import React from 'react';
import './App.sass';
import {List, Progress,} from 'antd';
import {CalendarOutlined,} from "@ant-design/icons";
import {Link} from "react-router-dom";

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

function App() {
  return (
    <div className="app objectives">
      <h1>All Objectives</h1>
      <h2>In progress</h2>
      <List
        className="list"
        dataSource={data}
        renderItem={(item, index) => (
          <List.Item>
            <List.Item.Meta
              title={<><Link to={`obj/${item.id}`}>{item.title}</Link></>}
              description={
                <>
                  <div><CalendarOutlined/> 2023/7/9 ~ 2023/8/9</div>
                  <Progress percent={30}/>
                </>
              }
            />
          </List.Item>
        )}
      />
      <div><Link to="obj">Create objective</Link></div>
    </div>
  )
}

export default App;
