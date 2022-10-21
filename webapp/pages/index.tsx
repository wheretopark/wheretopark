import {encodeParkingLots, parseParkingLots} from '../lib/types'
import {storekeeperClient} from '../lib/client'
import {GetStaticPropsContext, GetStaticPropsResult} from "next";
import {Home} from "../components/Home";

type IndexProps = {
    parkingLots: any,
}

const Index = ({parkingLots: parkingLotsJSON}: IndexProps) => {
    const parkingLots = parseParkingLots(JSON.stringify(parkingLotsJSON))
    return (
        <>
            <Home parkingLots={parkingLots}/>
        </>
    )
}


export async function getStaticProps(): Promise<GetStaticPropsResult<IndexProps>> {
    const parkingLots = encodeParkingLots(await storekeeperClient.parkingLots());
    const props: IndexProps = {
        parkingLots: JSON.parse(parkingLots),
    }
    return {
        props, // will be passed to the page component as props
        revalidate: 10
    }
}

export default Index
